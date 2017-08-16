package cache

import (
	"time"

	kitmetrics "github.com/go-kit/kit/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tapglue/snaas/platform/metrics"
)

type instrumentCountCache struct {
	component string
	errCount  kitmetrics.Counter
	hitCount  kitmetrics.Counter
	next      CountService
	opCount   kitmetrics.Counter
	opLatency *prometheus.HistogramVec
	service   string
	store     string
}

func InstrumentCountServiceMiddleware(
	component, service, store string,
	errCount kitmetrics.Counter,
	hitCount kitmetrics.Counter,
	opCount kitmetrics.Counter,
	opLatency *prometheus.HistogramVec,
) CountServiceMiddleware {
	return func(next CountService) CountService {
		return &instrumentCountCache{
			component: component,
			errCount:  errCount,
			hitCount:  hitCount,
			next:      next,
			opCount:   opCount,
			opLatency: opLatency,
			service:   service,
			store:     store,
		}
	}
}

func (s *instrumentCountCache) Decr(ns, key string) (count int, err error) {
	defer func(begin time.Time) {
		s.track("Decr", ns, begin, err)
	}(time.Now())

	return s.next.Decr(ns, key)
}

func (s *instrumentCountCache) Get(ns, key string) (count int, err error) {
	defer func(begin time.Time) {
		if err == nil {
			s.trackHit("Get", ns)
		}
		if IsKeyNotFound(err) {
			s.track("Get", ns, begin, nil)
			return
		}

		s.track("Get", ns, begin, err)
	}(time.Now())

	return s.next.Get(ns, key)
}

func (s *instrumentCountCache) Incr(ns, key string) (count int, err error) {
	defer func(begin time.Time) {
		s.track("Incr", ns, begin, err)
	}(time.Now())

	return s.next.Incr(ns, key)
}

func (s *instrumentCountCache) Set(ns, key string, count int) (err error) {
	defer func(begin time.Time) {
		s.track("Set", ns, begin, err)
	}(time.Now())

	return s.next.Set(ns, key, count)
}

func (s *instrumentCountCache) track(
	method, namespace string,
	begin time.Time,
	err error,
) {
	if err != nil {
		s.errCount.With(
			metrics.FieldComponent, s.component,
			metrics.FieldMethod, method,
			metrics.FieldNamespace, namespace,
			metrics.FieldService, s.service,
			metrics.FieldStore, s.store,
		).Add(1)

		return
	}

	s.opCount.With(
		metrics.FieldComponent, s.component,
		metrics.FieldMethod, method,
		metrics.FieldNamespace, namespace,
		metrics.FieldService, s.service,
		metrics.FieldStore, s.store,
	).Add(1)

	s.opLatency.With(prometheus.Labels{
		metrics.FieldComponent: s.component,
		metrics.FieldMethod:    method,
		metrics.FieldNamespace: namespace,
		metrics.FieldService:   s.service,
		metrics.FieldStore:     s.store,
	}).Observe(time.Since(begin).Seconds())
}

func (s *instrumentCountCache) trackHit(method, namespace string) {
	s.hitCount.With(
		metrics.FieldComponent, s.component,
		metrics.FieldMethod, method,
		metrics.FieldNamespace, namespace,
		metrics.FieldService, s.service,
		metrics.FieldStore, s.store,
	).Add(1)
}
