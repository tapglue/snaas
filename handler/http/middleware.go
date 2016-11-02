package http

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"

	"github.com/tapglue/api/platform/limiter"
	"github.com/tapglue/api/platform/metrics"
	"github.com/tapglue/api/service/app"
	"github.com/tapglue/api/service/session"
	"github.com/tapglue/api/service/user"
)

const headerIDFV = "X-Tapglue-Idfv"

var defaultEnabled = true

// CORS adds the standard set of CORS headers.
func CORS() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "User-Agent, Content-Type, Content-Length, Accept-Encoding, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(ctx, w, r)
		}
	}
}

// CtxApp extracts the App from the Authentication header.
func CtxApp(apps app.Service) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			token, _, ok := r.BasicAuth()
			if !ok {
				respondError(w, 1001, wrapError(ErrUnauthorized, "application user not found"))
				return
			}

			var currentApp *app.App

			if len(token) == 32 {
				as, err := apps.Query(app.NamespaceDefault, app.QueryOptions{
					Enabled: &defaultEnabled,
					Tokens: []string{
						token,
					},
				})
				if err != nil {
					respondError(w, 0, err)
					return
				}

				if len(as) == 1 {
					currentApp = as[0]
				}

				ctx = tokenTypeInContext(ctx, tokenApplication)
			} else if len(token) == 44 {
				as, err := apps.Query(app.NamespaceDefault, app.QueryOptions{
					BackendTokens: []string{
						token,
					},
					Enabled: &defaultEnabled,
				})
				if err != nil {
					respondError(w, 0, err)
					return
				}

				if len(as) == 1 {
					currentApp = as[0]
				}

				ctx = tokenTypeInContext(ctx, tokenBackend)
			}

			if currentApp == nil {
				respondError(w, 1001, wrapError(ErrUnauthorized, "application not found"))
				return
			}

			next(appInContext(ctx, currentApp), w, r)
		}
	}
}

// CtxPrepare adds a baseline of information to the Context currently:
// * api version
// * route name
func CtxPrepare(version string) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			route := "unknown"

			if current := mux.CurrentRoute(r); current != nil {
				route = current.GetName()
			}

			ctx = routeInContext(ctx, route)
			ctx = versionInContext(ctx, version)

			next(ctx, w, r)
		}
	}
}

// CtxUser extracts the user from the Authentication header and adds it to the
// Context.
func CtxUser(sessions session.Service, users user.Service) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			var (
				app       = appFromContext(ctx)
				tokenType = tokenTypeFromContext(ctx)
			)

			_, token, ok := r.BasicAuth()
			if !ok {
				respondError(w, 4007, wrapError(ErrUnauthorized, "error while reading user credentials"))
				return
			}

			if token == "" {
				respondError(w, 4013, wrapError(ErrUnauthorized, "session token missing from request"))
				return
			}

			var id uint64

			switch tokenType {
			case tokenApplication:
				ss, err := sessions.Query(app.Namespace(), session.QueryOptions{
					Enabled: &defaultEnabled,
					IDs:     []string{token},
				})
				if err != nil {
					respondError(w, 0, err)
					return
				}

				if len(ss) != 1 {
					respondError(w, 4007, wrapError(ErrUnauthorized, "invalid session token"))
					return
				}

				ctx = tokenInContext(ctx, token)
				id = ss[0].UserID
			case tokenBackend:
				var err error

				id, err = strconv.ParseUint(token, 10, 64)
				if err != nil {
					respondError(w, 0, err)
					return
				}
			default:
				respondError(w, 4007, wrapError(ErrUnauthorized, "error while reading user credentials"))
				return
			}

			us, err := users.Query(app.Namespace(), user.QueryOptions{
				Enabled: &defaultEnabled,
				IDs: []uint64{
					id,
				},
			})
			if err != nil {
				respondError(w, 0, err)
				return
			}

			if len(us) != 1 {
				respondError(w, 4007, wrapError(ErrUnauthorized, "user not found"))
				return
			}

			next(userInContext(ctx, us[0]), w, r)
		}
	}
}

// DebugHeaders adds extra information encoded in a custom header namespace for
// potential tracing and debugging post-mortem.
func DebugHeaders(rev, host string) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Tapglue-Host", host)
			w.Header().Set("X-Tapglue-Revision", rev)

			next(ctx, w, r)
		}
	}
}

// CtxDeviceID extracts the unique identification for a device.
func CtxDeviceID() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			deviceID := r.Header.Get(headerIDFV)

			if deviceID == "" {
				deviceID = session.DeviceIDUnknown
			}

			next(deviceIDInContext(ctx, deviceID), w, r)
		}
	}
}

// Gzip ensures proper encoding of the response if the client accepts it.
func Gzip() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				w.Header().Set("Content-Encoding", "gzip")

				gz := gzip.NewWriter(w)
				defer gz.Close()

				w = gzipResponseWriter{w, gz}
			}

			next(ctx, w, r)
		}
	}
}

// HasUserAgent ensures a valid User-Agent is set.
func HasUserAgent() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("User-Agent") == "" {
				respondError(w, 5002, wrapError(ErrBadRequest, "User-Agent header must be set"))
				return
			}

			next(ctx, w, r)
		}
	}
}

// Instrument observes key aspects of a request/response and exposes Prometheus
// metrics.
func Instrument(
	component string,
) Middleware {
	var (
		namespace         = "handler"
		subsystemRequest  = "request"
		subsystemResponse = "response"
		fieldKeys         = []string{
			metrics.FieldComponent,
			metrics.FieldVersion,
			metrics.FieldRoute,
			metrics.FieldStatus,
		}
		requestCount = kitprometheus.NewCounterFrom(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystemRequest,
			Name:      "count",
			Help:      "Number of requests received",
		}, fieldKeys)
		requestLatency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystemRequest,
				Name:      "latency_seconds",
				Help:      "Total duration of requests in seconds",
			},
			fieldKeys,
		)
		responseBytes = kitprometheus.NewCounterFrom(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystemResponse,
			Name:      "bytes",
			Help:      "Bytes returned as response bodies",
		}, fieldKeys)
	)

	prometheus.MustRegister(requestLatency)

	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			var (
				begin     = time.Now()
				resr      = newResponseRecorder(w)
				routeName = routeFromContext(ctx)
				version   = versionFromContext(ctx)
			)

			next(ctx, resr, r)

			var (
				status = strconv.Itoa(resr.statusCode)
			)

			requestCount.With(
				metrics.FieldComponent, component,
				metrics.FieldRoute, routeName,
				metrics.FieldStatus, status,
				metrics.FieldVersion, version,
			).Add(1)
			responseBytes.With(
				metrics.FieldComponent, component,
				metrics.FieldRoute, routeName,
				metrics.FieldStatus, status,
				metrics.FieldVersion, version,
			).Add(float64(resr.contentLength))
			requestLatency.With(prometheus.Labels{
				metrics.FieldComponent: component,
				metrics.FieldRoute:     routeName,
				metrics.FieldStatus:    status,
				metrics.FieldVersion:   version,
			}).Observe(time.Since(begin).Seconds())
		}
	}
}

// Log logs information per single request-response.
func Log(logger log.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			var (
				begin   = time.Now()
				reqr    = newRequestRecorder(r)
				resr    = newResponseRecorder(w)
				route   = routeFromContext(ctx)
				version = versionFromContext(ctx)
			)

			next(ctx, resr, r)

			logger.Log(
				"duration_ns", time.Since(begin).Nanoseconds(),
				"query", r.URL.Query(),
				"request", reqr,
				"response", resr,
				"route", route,
				"version", version,
			)
		}
	}
}

// RateLimit enforces request limits per application.
func RateLimit(limits limiter.Limiter) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			var (
				app = appFromContext(ctx)
				l   = &limiter.Limitee{
					Hash:       app.Token,
					Limit:      app.Limit(),
					WindowSize: time.Minute,
				}
			)

			quota, expires, err := limits.Request(l)
			if err != nil {
				respondError(w, 0, err)
				return
			}

			w.Header().Set("X-Ratelimit-Quota", strconv.FormatInt(app.Limit(), 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(quota, 10))
			w.Header().Set("X-Ratelimit-Reset", strconv.FormatInt(expires.Unix(), 10))

			if quota < 0 {
				respondError(w, 0, wrapError(ErrLimitExceeded, "request quota exceeded"))
				return
			}

			next(ctx, w, r)
		}
	}
}

// SecureHeaders adds a list of commonly recgonised best-pratice security
// headers.
// Source: https://www.owasp.org/index.php/List_of_useful_HTTP_headers
func SecureHeaders() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")

			next(ctx, w, r)
		}
	}
}

// ValidateContent checks if content-length and content-type are set for
// requests with paylaod and adhere to our required limits and values.
func ValidateContent() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" && r.Method != "PUT" {
				next(ctx, w, r)
				return
			}

			if cl := r.Header.Get("Content-Length"); cl == "" {
				respondError(w, 5004, wrapError(ErrBadRequest, "Content-Length header missing"))
				return
			} else if l, err := strconv.ParseInt(cl, 10, 64); err != nil {
				respondError(w, 5003, wrapError(ErrBadRequest, "Content-Length header is invalid"))
				return
			} else if l != r.ContentLength {
				respondError(w, 5005, wrapError(ErrBadRequest, "Content-Length header size mismatch"))
				return
			} else if r.ContentLength > 32768 {
				respondError(w, 5011, wrapError(ErrBadRequest, "payload too big"))
				return
			}

			// Enforece content type checking for requests with payload.
			if r.ContentLength > 0 {
				if ct := r.Header.Get("Content-Type"); ct == "" {
					respondError(w, 5007, wrapError(ErrBadRequest, "Content-Type header missing"))
					return
				} else if ct != "application/json" && ct != "application/json; charset=UTF-8" {
					respondError(w, 5006, wrapError(ErrBadRequest, "Content-Type header missmatch"))
					return
				}
			}

			if r.Body == nil {
				respondError(w, 5012, wrapError(ErrBadRequest, "empty request body"))
				return
			}

			next(ctx, w, r)
		}
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	io.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

type requestRecorder struct {
	Header           map[string][]string `json:"header"`
	Host             string              `json:"host"`
	Method           string              `json:"method"`
	Proto            string              `json:"proto"`
	RemoteAddr       string              `json:"remoteAddr"`
	RequestURI       string              `json:"requestURI"`
	TransferEncoding []string            `json:"transferEncoding"`
	URL              string              `json:"url"`
}

func newRequestRecorder(r *http.Request) *requestRecorder {
	return &requestRecorder{
		Header:           r.Header,
		Host:             r.Host,
		Method:           strings.ToLower(r.Method),
		Proto:            r.Proto,
		RemoteAddr:       r.RemoteAddr,
		RequestURI:       r.RequestURI,
		TransferEncoding: r.TransferEncoding,
		URL:              r.URL.String(),
	}
}

type responseRecorder struct {
	http.ResponseWriter `json:"-"`

	contentLength int
	statusCode    int
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
	}
}

func (rc *responseRecorder) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ContentLength int                 `json:"contentLength"`
		Headers       map[string][]string `json:"header"`
		StatusCode    int                 `json:"statusCode"`
	}{
		ContentLength: rc.contentLength,
		Headers:       rc.ResponseWriter.Header(),
		StatusCode:    rc.statusCode,
	})
}

func (rc *responseRecorder) Write(b []byte) (int, error) {
	n, err := rc.ResponseWriter.Write(b)

	rc.contentLength += n

	return n, err
}

func (rc *responseRecorder) WriteHeader(code int) {
	rc.statusCode = code
	rc.ResponseWriter.WriteHeader(code)
}
