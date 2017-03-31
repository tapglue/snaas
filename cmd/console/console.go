package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/tapglue/snaas/core"
	handler "github.com/tapglue/snaas/handler/http"
	"github.com/tapglue/snaas/platform/cache"
	"github.com/tapglue/snaas/platform/generate"
	"github.com/tapglue/snaas/platform/metrics"
	"github.com/tapglue/snaas/platform/redis"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/device"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/rule"
	"github.com/tapglue/snaas/service/user"
)

const (
	component = "console"

	namespaceCache   = "cache"
	namespaceService = "service"

	serviceObjectCounts = "object_counts"
	subsystemHit        = "hit"
	storeCache          = "redis"
	storeService        = "postgres"

	version = "0.4"
)

// Timeouts
const (
	defaultReadTimeout  = 2 * time.Second
	defaultWriteTimeout = 3 * time.Second
)

// Buildtime variables.
var (
	revision = "0000000-dev"
)

// Templates.
var (
	tmplIndex = `<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">
    <link href="https://fonts.googleapis.com/css?family=Fira+Sans:300,300i,400,500,700" rel="stylesheet">
    <link href="/styles/normalize.css" rel="stylesheet">
    <link href="/styles/nucleo-glyph.css" rel="stylesheet">
    <link href="/styles/nucleo-outline.css" rel="stylesheet">
    <link href="/styles/console.css" rel="stylesheet">
    <script src="/scripts/console.js" type="text/javascript"></script>
  </head>
  <body>
    <script type="text/javascript">
		var app = Elm.Main.fullscreen({
			loginUrl: "{{.LoginURL}}",
			zone: "{{.Zone}}"
		});
    </script>
 </body>
</html>`
)

func main() {
	var (
		begin = time.Now()

		env                = flag.String("env", "dev", "Environment used for isolation.")
		googleClientID     = flag.String("google.client.id", "", "Google OAuth client identifier")
		googleClientSecret = flag.String("google.client.secret", "", "Google OAuth client secret")
		googleCallback     = flag.String("google.callback", "http://localhost:8084/oauth2callback", "URL to return to from the auth flow")
		listenAddr         = flag.String("listen.adrr", ":8084", "HTTP bind address for main API")
		postgresURL        = flag.String("postgres.url", "", "Postgres URL to connect to")
		redisAddr          = flag.String("redis.addr", ":6379", "Redis address to connect to")
		region             = flag.String("region", "local", "AWS region of the current deployment")
		staticLocal        = flag.Bool("static.local", false, "Determines if static files are loaded from the filesystem")
		telemetryAddr      = flag.String("telemetry.addr", ":9002", "HTTP bind address where telemetry is exposed")
	)
	flag.Parse()

	// Setup logging.
	logger := log.With(
		log.NewJSONLogger(os.Stdout),
		"caller", log.Caller(3),
		"component", component,
		"revision", revision,
	)

	hostname, err := os.Hostname()
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
		os.Exit(1)
	}

	logger = log.With(logger, "host", hostname)

	// Setup instrumentation.
	go func(addr string) {
		logger.Log(
			"duration", time.Now().Sub(begin).Nanoseconds(),
			"lifecycle", "start",
			"listen", addr,
			"sub", "telemetry",
		)

		http.Handle("/metrics", prometheus.Handler())

		err := http.ListenAndServe(addr, nil)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort", "sub", "telemetry")
			os.Exit(1)
		}
	}(*telemetryAddr)

	cacheFieldKeys := []string{
		metrics.FieldComponent,
		metrics.FieldMethod,
		metrics.FieldNamespace,
		metrics.FieldService,
		metrics.FieldStore,
	}

	cacheErrCount, cacheOpCount, cacheOpLatency := metrics.KeyMetrics(
		namespaceCache,
		cacheFieldKeys...,
	)

	cacheHitCount := kitprometheus.NewCounterFrom(prometheus.CounterOpts{
		Namespace: namespaceCache,
		Subsystem: subsystemHit,
		Name:      "count",
		Help:      "Number of cache hits",
	}, cacheFieldKeys)

	serviceErrCount, serviceOpCount, serviceOpLatency := metrics.KeyMetrics(
		namespaceService,
		metrics.FieldComponent,
		metrics.FieldMethod,
		metrics.FieldNamespace,
		metrics.FieldService,
		metrics.FieldStore,
	)

	// Setup clients.
	authConf := &oauth2.Config{
		ClientID:     *googleClientID,
		ClientSecret: *googleClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  *googleCallback,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}

	pgClient, err := sqlx.Connect(storeService, *postgresURL)
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
		os.Exit(1)
	}

	redisPool := redis.Pool(*redisAddr, "")

	// Setup caches.
	var objectCountsCache cache.CountService
	objectCountsCache = cache.RedisCountService(redisPool)
	objectCountsCache = cache.InstrumentCountServiceMiddleware(
		component,
		serviceObjectCounts,
		storeCache,
		cacheErrCount,
		cacheHitCount,
		cacheOpCount,
		cacheOpLatency,
	)(objectCountsCache)

	// Setup services.
	var apps app.Service
	apps = app.PostgresService(pgClient)
	apps = app.InstrumentServiceMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(apps)
	apps = app.LogServiceMiddleware(logger, storeService)(apps)

	var connections connection.Service
	connections = connection.PostgresService(pgClient)
	connections = connection.InstrumentServiceMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(connections)
	connections = connection.LogServiceMiddleware(logger, storeService)(connections)

	var devices device.Service
	devices = device.PostgresService(pgClient)
	devices = device.InstrumentServiceMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(devices)
	devices = device.LogServiceMiddleware(logger, storeService)(devices)

	var objects object.Service
	objects = object.PostgresService(pgClient)
	objects = object.InstrumentServiceMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(objects)
	objects = object.LogServiceMiddleware(logger, storeService)(objects)
	objects = object.CacheServiceMiddleware(objectCountsCache)(objects)

	var rules rule.Service
	rules = rule.PostgresService(pgClient)
	rules = rule.InstrumentServiceMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(rules)

	var users user.Service
	users = user.PostgresService(pgClient)
	users = user.InstrumentMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(users)
	users = user.LogMiddleware(logger, storeService)(users)

	// Setup middlewares.
	var (
		withConstraints = handler.Chain(
			handler.CtxPrepare(version),
			handler.Log(logger),
			handler.Instrument(component),
			handler.SecureHeaders(),
			handler.DebugHeaders(revision, hostname),
			handler.CORS(),
			handler.HasUserAgent(),
		)
	)

	// Setup templates.
	tplRoot, err := template.New("root").Parse(tmplIndex)

	// Setup Router.
	router := mux.NewRouter()

	router.Methods("GET").Path(`/health-45016490610398192`).Name("healthcheck").HandlerFunc(
		handler.Wrap(
			handler.CtxPrepare(version),
			handler.Health(pgClient, redisPool),
		),
	)

	router.Methods("GET").Path("/api/apps/{appID:[0-9]+}").Name("appRetrieve").HandlerFunc(
		handler.Wrap(
			withConstraints,
			handler.AppRetrieve(
				core.AppFetchWithCounts(
					apps,
					connections,
					devices,
					objects,
					rules,
					users,
				),
			),
		),
	)

	router.Methods("GET").Path("/api/apps").Name("appList").HandlerFunc(
		handler.Wrap(
			withConstraints,
			handler.AppList(core.AppList(apps)),
		),
	)

	router.Methods("POST").Path("/api/apps").Name("appCreate").HandlerFunc(
		handler.Wrap(
			withConstraints,
			handler.AppCreate(core.AppCreate(apps)),
		),
	)

	router.Methods("PUT").Path("/api/apps/{appID:[0-9]+}/rules/{ruleID:[0-9]+}/activate").Name("ruleDeactivate").HandlerFunc(
		handler.Wrap(
			withConstraints,
			handler.RuleActivate(core.RuleActivate(apps, rules)),
		),
	)

	router.Methods("PUT").Path("/api/apps/{appID:[0-9]+}/rules/{ruleID:[0-9]+}/deactivate").Name("ruleDeactivate").HandlerFunc(
		handler.Wrap(
			withConstraints,
			handler.RuleDeactivate(core.RuleDeactivate(apps, rules)),
		),
	)

	router.Methods("DELETE").Path("/api/apps/{appID:[0-9]+}/rules/{ruleID:[0-9]+}").Name("ruleDelete").HandlerFunc(
		handler.Wrap(
			withConstraints,
			handler.RuleDelete(core.RuleDelete(apps, rules)),
		),
	)

	router.Methods("GET").Path("/api/apps/{appID:[0-9]+}/rules/{ruleID:[0-9]+}").Name("ruleRetrieve").HandlerFunc(
		handler.Wrap(
			withConstraints,
			handler.RuleRetrieve(core.RuleFetch(apps, rules)),
		),
	)

	router.Methods("GET").Path("/api/apps/{appID:[0-9]+}/rules").Name("ruleList").HandlerFunc(
		handler.Wrap(
			withConstraints,
			handler.RuleList(core.RuleList(apps, rules)),
		),
	)

	router.Methods("POST").Path("/api/me").Name("memberRetrieveMe").HandlerFunc(
		handler.Wrap(
			withConstraints,
			handler.MemberRetrieveMe(authConf),
		),
	)

	router.Methods("POST").Path("/api/me/login").Name("memberLogin").HandlerFunc(
		handler.Wrap(
			withConstraints,
			handler.MemberLogin(authConf),
		),
	)

	router.Methods("GET").PathPrefix("/fonts").Name("fonts").Handler(
		http.FileServer(FS(*staticLocal)),
	)

	router.Methods("GET").PathPrefix("/scripts").Name("scripts").Handler(
		http.FileServer(FS(*staticLocal)),
	)

	router.Methods("GET").PathPrefix("/styles").Name("styles").Handler(
		http.FileServer(FS(*staticLocal)),
	)

	router.Methods("GET").PathPrefix("/").Name("root").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api") {
				http.NotFound(w, r)
				return
			}

			tplRoot.Execute(w, struct {
				LoginURL string
				Zone     string
			}{
				LoginURL: authConf.AuthCodeURL(generateID()),
				Zone:     fmt.Sprintf("%s-%s", *env, *region),
			})
		},
	)

	// Setup server.
	server := &http.Server{
		Addr:         *listenAddr,
		Handler:      router,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
	}

	logger.Log(
		"duration", time.Now().Sub(begin).Nanoseconds(),
		"lifecycle", "start",
		"listen", *listenAddr,
		"sub", "api",
	)

	err = server.ListenAndServe()
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort", "sub", "api")
		os.Exit(1)
	}
}

func generateID() string {
	src := rand.NewSource(time.Now().UnixNano())

	return base64.StdEncoding.EncodeToString(generate.RandomBytes(src, 24))
}
