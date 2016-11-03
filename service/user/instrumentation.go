package user

import (
	"time"

	kitmetrics "github.com/go-kit/kit/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tapglue/snaas/platform/metrics"
)

const serviceName = "user"

type instrumentService struct {
	component string
	errCount  kitmetrics.Counter
	next      Service
	opCount   kitmetrics.Counter
	opLatency *prometheus.HistogramVec
	store     string
}

// InstrumentMiddleware observes key apsects of Service operations and exposes
// Prometheus metrics.
func InstrumentMiddleware(
	component, store string,
	errCount kitmetrics.Counter,
	opCount kitmetrics.Counter,
	opLatency *prometheus.HistogramVec,
) ServiceMiddleware {
	return func(next Service) Service {
		return &instrumentService{
			component: component,
			errCount:  errCount,
			opCount:   opCount,
			opLatency: opLatency,
			next:      next,
			store:     store,
		}
	}
}

func (s *instrumentService) Count(
	ns string,
	opts QueryOptions,
) (count int, err error) {
	defer func(begin time.Time) {
		s.track("Count", ns, begin, err)
	}(time.Now())

	return s.next.Count(ns, opts)
}

func (s *instrumentService) Put(
	ns string,
	input *User,
) (output *User, err error) {
	defer func(begin time.Time) {
		s.track("Put", ns, begin, err)
	}(time.Now())

	return s.next.Put(ns, input)
}

func (s *instrumentService) PutLastRead(
	ns string,
	userID uint64,
	ts time.Time,
) (err error) {
	defer func(begin time.Time) {
		s.track("PutLastRead", ns, begin, err)
	}(time.Now())

	return s.next.PutLastRead(ns, userID, ts)
}

func (s *instrumentService) Query(
	ns string,
	opts QueryOptions,
) (list List, err error) {
	defer func(begin time.Time) {
		s.track("Query", ns, begin, err)
	}(time.Now())

	return s.next.Query(ns, opts)
}

func (s *instrumentService) Search(
	ns string,
	opts QueryOptions,
) (list List, err error) {
	defer func(begin time.Time) {
		s.track("Search", ns, begin, err)
	}(time.Now())

	return s.next.Search(ns, opts)
}

func (s *instrumentService) Setup(ns string) (err error) {
	defer func(begin time.Time) {
		s.track("Setup", ns, begin, err)
	}(time.Now())

	return s.next.Setup(ns)
}

func (s *instrumentService) Teardown(ns string) (err error) {
	defer func(begin time.Time) {
		s.track("Teardown", ns, begin, err)
	}(time.Now())

	return s.next.Teardown(ns)
}

func (s *instrumentService) track(
	method, namespace string,
	begin time.Time,
	err error,
) {
	if err != nil {
		s.errCount.With(
			metrics.FieldComponent, s.component,
			metrics.FieldMethod, method,
			metrics.FieldNamespace, namespace,
			metrics.FieldService, serviceName,
			metrics.FieldStore, s.store,
		).Add(1)
	}

	s.opCount.With(
		metrics.FieldComponent, s.component,
		metrics.FieldMethod, method,
		metrics.FieldNamespace, namespace,
		metrics.FieldService, serviceName,
		metrics.FieldStore, s.store,
	).Add(1)

	s.opLatency.With(prometheus.Labels{
		metrics.FieldComponent: s.component,
		metrics.FieldMethod:    method,
		metrics.FieldNamespace: namespace,
		metrics.FieldService:   serviceName,
		metrics.FieldStore:     s.store,
	}).Observe(time.Since(begin).Seconds())
}
