package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tapglue/snaas/core"
	handler "github.com/tapglue/snaas/handler/http"
	"github.com/tapglue/snaas/platform/cache"
	"github.com/tapglue/snaas/platform/limiter"
	"github.com/tapglue/snaas/platform/metrics"
	"github.com/tapglue/snaas/platform/redis"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/device"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/session"
	"github.com/tapglue/snaas/service/user"
)

// Logging and telemetry identifiers.
const (
	component           = "gateway-http"
	namespaceCache      = "cache"
	namespaceService    = "service"
	namespaceSource     = "source"
	subsystemHit        = "hit"
	subsystemQueue      = "queue"
	serviceEventCounts  = "event_counts"
	serviceObjectCounts = "object_counts"
	storeCache          = "redis"
	storeService        = "postgres"
)

// Versions.
const (
	versionCurrent = "0.4"
)

// Supported source types.
const (
	sourceNop = "nop"
	sourceSQS = "sqs"
)

// Prefixes.
const (
	prefixRateLimiter = "ratelimiter:app:"
)

// Timeouts
const (
	defaultReadTimeout  = 2 * time.Second
	defaultWriteTimeout = 3 * time.Second
)

// Buildtime vars.
var (
	revision = "0000000-dev"
)

func main() {
	var (
		begin = time.Now()

		awsID         = flag.String("aws.id", "", "Identifier for AWS requests")
		awsRegion     = flag.String("aws.region", "us-east-1", "AWS Region to operate in")
		awsSecret     = flag.String("aws.secret", "", "Identification secret for AWS requests")
		listenAddr    = flag.String("listen.addr", ":8083", "HTTP bind address for main API")
		postgresURL   = flag.String("postgres.url", "", "Postgres URL to connect to")
		redisAddr     = flag.String("redis.addr", ":6379", "Redis address to connect to")
		source        = flag.String("source", sourceNop, "Source type used for state change propagations")
		telemetryAddr = flag.String("telemetry.addr", ":9000", "HTTP bind address where prometheus telemetry is exposed")
	)
	flag.Parse()

	// Setup logging.
	logger := log.NewContext(
		log.NewJSONLogger(os.Stdout),
	).With(
		"caller", log.Caller(3),
		"component", component,
		"revision", revision,
	)

	hostname, err := os.Hostname()
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
	}

	logger = log.NewContext(logger).With("host", hostname)

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

	sourceFieldKeys := []string{
		metrics.FieldComponent,
		metrics.FieldMethod,
		metrics.FieldNamespace,
		metrics.FieldSource,
		metrics.FieldStore,
	}

	sourceErrCount, sourceOpCount, sourceOpLatency := metrics.KeyMetrics(
		namespaceSource,
		sourceFieldKeys...,
	)

	sourceQueueLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespaceSource,
			Subsystem: subsystemQueue,
			Name:      "latency_seconds",
			Help:      "Distribution of message queue latency in seconds",
			Buckets:   metrics.BucketsQueue,
		},
		sourceFieldKeys,
	)
	prometheus.MustRegister(sourceQueueLatency)

	// Setup clients.
	var (
		aSession = awsSession.New(&aws.Config{
			Credentials: credentials.NewStaticCredentials(*awsID, *awsSecret, ""),
			Region:      aws.String(*awsRegion),
		})
		redisPool   = redis.Pool(*redisAddr, "")
		rateLimiter = limiter.Redis(redisPool, prefixRateLimiter)
		sqsAPI      = sqs.New(aSession)
	)

	pgClient, err := sqlx.Connect(storeService, *postgresURL)
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
		os.Exit(1)
	}

	// Setup caches
	var eventCountsCache cache.CountService
	eventCountsCache = cache.RedisCountService(redisPool)
	eventCountsCache = cache.InstrumentCountServiceMiddleware(
		component,
		serviceEventCounts,
		storeCache,
		cacheErrCount,
		cacheHitCount,
		cacheOpCount,
		cacheOpLatency,
	)(eventCountsCache)

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

	// Setup sources.
	var (
		conSource    connection.Source
		eventSource  event.Source
		objectSource object.Source
	)

	switch *source {
	case sourceNop:
		conSource = connection.NopSource()
		eventSource = event.NopSource()
		objectSource = object.NopSource()
	case sourceSQS:
		conSource, err = connection.SQSSource(sqsAPI)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort")
			os.Exit(1)
		}

		eventSource, err = event.SQSSource(sqsAPI)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort")
			os.Exit(1)
		}

		objectSource, err = object.SQSSource(sqsAPI)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort")
			os.Exit(1)
		}
	default:
		logger.Log(
			"err", fmt.Sprintf("Source type '%s' not supported", *source),
			"lifecycle", "abort",
		)
		os.Exit(1)
	}

	conSource = connection.InstrumentSourceMiddleware(
		component,
		*source,
		sourceErrCount,
		sourceOpCount,
		sourceOpLatency,
		sourceQueueLatency,
	)(conSource)
	conSource = connection.LogSourceMiddleware(*source, logger)(conSource)

	eventSource = event.InstrumentSourceMiddleware(
		component,
		*source,
		sourceErrCount,
		sourceOpCount,
		sourceOpLatency,
		sourceQueueLatency,
	)(eventSource)
	eventSource = event.LogSourceMiddleware(*source, logger)(eventSource)

	objectSource = object.InstrumentSourceMiddleware(
		component,
		*source,
		sourceErrCount,
		sourceOpCount,
		sourceOpLatency,
		sourceQueueLatency,
	)(objectSource)
	objectSource = object.LogSourceMiddleware(*source, logger)(objectSource)

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
	// Combine connection service and source.
	connections = connection.SourcingServiceMiddleware(conSource)(connections)

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

	var events event.Service
	events = event.PostgresService(pgClient)
	events = event.InstrumentServiceMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(events)
	events = event.LogServiceMiddleware(logger, storeService)(events)
	// Combine event service and source.
	events = event.SourcingServiceMiddleware(eventSource)(events)
	// Wrap service with caching.
	// TODO: Implement write path to avoid stale counts.
	// events = event.CacheServiceMiddleware(eventCountsCache)(events)

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
	// Combine object service and source.
	objects = object.SourcingServiceMiddleware(objectSource)(objects)
	// Wrap service with caching
	// TODO: Implement write path to avoid stale counts.
	// objects = object.CacheServiceMiddleware(objectCountsCache)(objects)

	var sessions session.Service
	sessions = session.PostgresService(pgClient)
	sessions = session.InstrumentMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(sessions)
	sessions = session.LogMiddleware(logger, storeService)(sessions)

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
		withApp = handler.Chain(
			handler.CtxPrepare(versionCurrent),
			handler.Log(logger),
			handler.Instrument(component),
			handler.SecureHeaders(),
			handler.DebugHeaders(revision, hostname),
			handler.CORS(),
			handler.Gzip(),
			handler.HasUserAgent(),
			handler.ValidateContent(),
			handler.CtxApp(apps),
			handler.CtxDeviceID(),
			handler.RateLimit(rateLimiter),
		)
		withUser = handler.Chain(
			withApp,
			handler.CtxUser(sessions, users),
		)
	)

	// Setup Router.
	router := mux.NewRouter().StrictSlash(true)

	router.Methods("GET").Path(`/health-45016490610398192`).Name("healthcheck").HandlerFunc(
		handler.Wrap(
			handler.CtxPrepare(versionCurrent),
			handler.Health(pgClient, redisPool),
		),
	)

	current := router.PathPrefix(fmt.Sprintf("/%s", versionCurrent)).Subrouter()

	// Connection routes.
	current.Methods("GET").Path(`/me/connections/{state:[a-z]+}`).Name("connectionListByState").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.ConnectionByState(
				core.ConnectionByState(connections, users),
			),
		),
	)

	current.Methods("DELETE").Path(`/me/connections/{type:[a-z]+}/{toID:[0-9]+}`).Name("connectionDelete").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.ConnectionDelete(
				core.ConnectionDelete(connections),
			),
		),
	)

	current.Methods("POST").Path(`/me/connections/social`).Name("connectionCreateSocial").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.ConnectionSocial(
				core.ConnectionCreateSocial(connections, users),
			),
		),
	)

	current.Methods("PUT").Path(`/me/connections`).Name("connectionUpdate").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.ConnectionUpdate(
				core.ConnectionUpdate(connections, users),
			),
		),
	)

	current.Methods("GET").Path(`/me/followers`).Name("connectionFollowersMe").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.ConnectionFollowersMe(
				core.ConnectionFollowers(connections, users),
			),
		),
	)

	current.Methods("GET").Path(`/me/follows`).Name("connectionFollowingsMe").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.ConnectionFollowingsMe(
				core.ConnectionFollowings(connections, users),
			),
		),
	)

	current.Methods("GET").Path(`/me/friends`).Name("connectionFriendsMe").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.ConnectionFriendsMe(
				core.ConnectionFriends(connections, users),
			),
		),
	)

	current.Methods("GET").Path(`/users/{userID:[0-9]+}/followers`).Name("connectionFollowers").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.ConnectionFollowers(
				core.ConnectionFollowers(connections, users),
			),
		),
	)

	current.Methods("GET").Path(`/users/{userID:[0-9]+}/follows`).Name("connectionFollowings").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.ConnectionFollowings(
				core.ConnectionFollowings(connections, users),
			),
		),
	)

	current.Methods("GET").Path(`/users/{userID:[0-9]+}/friends`).Name("connectionFriends").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.ConnectionFriends(
				core.ConnectionFriends(connections, users),
			),
		),
	)

	// Device routes.
	current.Methods("DELETE").Path(`/me/devices/{deviceID}`).Name("deviceDelete").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.DeviceDelete(
				core.DeviceDelete(devices),
			),
		),
	)

	current.Methods("PUT").Path(`/me/devices/{deviceID}`).Name("deviceUpdate").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.DeviceUpdate(
				core.DeviceUpdate(devices),
			),
		),
	)

	// Feed routes.
	current.Methods("GET").Path("/me/feed").Name("feedNews").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.FeedNews(
				core.FeedNews(connections, events, objects, users),
			),
		),
	)

	current.Methods("GET").Path("/me/feed/events").Name("feedEvents").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.FeedEvents(
				core.FeedEvents(connections, events, objects, users),
			),
		),
	)

	current.Methods("GET").Path("/me/feed/notifications/self").Name("feedNotificationsSelf").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.FeedNotificationsSelf(
				core.FeedNotificationsSelf(connections, events, objects, users),
			),
		),
	)

	current.Methods("GET").Path("/me/feed/posts").Name("feedPosts").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.FeedPosts(
				core.FeedPosts(connections, events, objects, users),
			),
		),
	)

	// Post routes.
	current.Methods("POST").Path("/posts").Name("postCreate").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.PostCreate(
				core.PostCreate(objects),
			),
		),
	)

	current.Methods("DELETE").Path("/posts/{postID:[0-9]+}").Name("postDelete").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.PostDelete(
				core.PostDelete(objects),
			),
		),
	)

	current.Methods("GET").Path("/posts/{postID:[0-9]+}").Name("postRetrieve").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.PostRetrieve(
				core.PostRetrieve(connections, events, objects),
			),
		),
	)

	current.Methods("PUT").Path("/posts/{postID:[0-9]+}").Name("postUpdate").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.PostUpdate(
				core.PostUpdate(objects),
			),
		),
	)

	current.Methods("GET").Path("/posts").Name("postListAll").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.PostListAll(
				core.PostListAll(events, objects, users),
			),
		),
	)

	current.Methods("GET").Path("/me/posts").Name("postListMe").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.PostListMe(
				core.PostListUser(connections, events, objects, users),
			),
		),
	)

	current.Methods("GET").Path("/users/{userID:[0-9]+}/posts").Name("postList").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.PostList(
				core.PostListUser(connections, events, objects, users),
			),
		),
	)

	// Comment routes.
	current.Methods("POST").Path("/posts/{postID:[0-9]+}/comments").Name("commentCreate").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.CommentCreate(
				core.CommentCreate(connections, objects),
			),
		),
	)

	current.Methods("DELETE").Path("/posts/{postID:[0-9]+}/comments/{commentID:[0-9]+}").Name("commentDelete").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.CommentDelete(
				core.CommentDelete(connections, objects),
			),
		),
	)

	current.Methods("GET").Path("/posts/{postID:[0-9]+}/comments/{commentID:[0-9]+}").Name("commentRetrieve").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.CommentRetrieve(
				core.CommentRetrieve(objects),
			),
		),
	)

	current.Methods("PUT").Path("/posts/{postID:[0-9]+}/comments/{commentID:[0-9]+}").Name("commentUpdate").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.CommentUpdate(
				core.CommentUpdate(objects),
			),
		),
	)

	current.Methods("GET").Path("/posts/{postID:[0-9]+}/comments").Name("commentList").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.CommentList(
				core.CommentList(connections, objects, users),
			),
		),
	)

	// Like routes.
	current.Methods("POST").Path("/posts/{postID:[0-9]+}/likes").Name("likeCreate").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.LikeCreate(
				core.LikeCreate(connections, events, objects),
			),
		),
	)

	current.Methods("DELETE").Path("/posts/{postID:[0-9]+}/likes").Name("likeDelete").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.LikeDelete(
				core.LikeDelete(connections, events, objects),
			),
		),
	)

	current.Methods("GET").Path("/posts/{postID:[0-9]+}/likes").Name("likesPost").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.LikesPost(
				core.LikeList(connections, events, objects, users),
			),
		),
	)

	current.Methods("GET").Path("/me/likes").Name("likesMe").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.LikesMe(
				core.LikesUser(connections, events, objects, users),
			),
		),
	)

	current.Methods("GET").Path(`/users/{userID:[0-9]+}/likes`).HandlerFunc(
		handler.Wrap(
			withUser,
			handler.LikesUser(
				core.LikesUser(connections, events, objects, users),
			),
		),
	)

	// User routes.
	current.Methods("GET").Path("/me").Name("userRetrieveMe").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.UserRetrieveMe(
				core.UserRetrieve(connections, sessions, users),
			),
		),
	)

	current.Methods("PUT").Path("/me").Name("userUpdate").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.UserUpdate(
				core.UserUpdate(connections, sessions, users),
			),
		),
	)

	current.Methods("POST").Path("/me/login").Name("userLogin").HandlerFunc(
		handler.Wrap(
			withApp,
			handler.UserLogin(
				core.UserLogin(connections, sessions, users),
			),
		),
	)

	current.Methods("DELETE").Path("/me/logout").Name("userLogout").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.UserLogout(
				core.UserLogout(sessions),
			),
		),
	)

	current.Methods("DELETE").Path("/me").Name("userDelete").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.UserDelete(
				core.UserDelete(users),
			),
		),
	)

	current.Methods("GET").Path("/users/{userID:[0-9]+}").Name("userRetrieve").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.UserRetrieve(
				core.UserRetrieve(connections, sessions, users),
			),
		),
	)

	current.Methods("POST").Path("/users/search/emails").Name("userSearchEmails").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.UserSearchEmails(
				core.UserListByEmails(connections, users),
			),
		),
	)

	current.Methods("POST").Path(`/users/search/{platform:[a-z]+}`).Name("userSearchPlatform").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.UserSearchPlatform(
				core.UserListByPlatformIDs(connections, users),
			),
		),
	)

	current.Methods("GET").Path(`/users/search`).Name("userSearch").HandlerFunc(
		handler.Wrap(
			withUser,
			handler.UserSearch(
				core.UserSearch(connections, users),
			),
		),
	)

	current.Methods("POST").Path(`/users`).Name("userCreate").HandlerFunc(
		handler.Wrap(
			withApp,
			handler.UserCreate(
				core.UserCreate(sessions, users),
			),
		),
	)

	// Setup server.
	server := &http.Server{
		Addr:         *listenAddr,
		Handler:      router,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
	}

	// TODO: Handle TLS.

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
