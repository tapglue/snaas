package connection

import (
	"time"

	kitmetrics "github.com/go-kit/kit/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tapglue/snaas/platform/metrics"
)

const serviceName = "connection"

type instrumentService struct {
	component string
	errCount  kitmetrics.Counter
	opCount   kitmetrics.Counter
	opLatency *prometheus.HistogramVec
	next      Service
	store     string
}

// InstrumentServiceMiddleware observes key aspects of Service operations and
// exposes Prometheus metrics.
func InstrumentServiceMiddleware(
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

func (s *instrumentService) Friends(ns string, origin uint64) (ls List, err error) {
	defer func(begin time.Time) {
		s.track("Friends", ns, begin, err)
	}(time.Now())

	return s.next.Friends(ns, origin)
}

func (s *instrumentService) Put(
	ns string,
	input *Connection,
) (output *Connection, err error) {
	defer func(begin time.Time) {
		s.track("Put", ns, begin, err)
	}(time.Now())

	return s.next.Put(ns, input)
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
	method string,
	namespace string,
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

type instrumentSource struct {
	component    string
	errCount     kitmetrics.Counter
	opCount      kitmetrics.Counter
	opLatency    *prometheus.HistogramVec
	queueLatency *prometheus.HistogramVec
	next         Source
	store        string
}

// InstrumentSourceMiddleware observes key aspects of Source operations and
// exposes Prometheus metrics.
func InstrumentSourceMiddleware(
	component, store string,
	errCount kitmetrics.Counter,
	opCount kitmetrics.Counter,
	opLatency *prometheus.HistogramVec,
	queueLatency *prometheus.HistogramVec,
) SourceMiddleware {
	return func(next Source) Source {
		return &instrumentSource{
			component:    component,
			errCount:     errCount,
			opCount:      opCount,
			opLatency:    opLatency,
			queueLatency: queueLatency,
			next:         next,
			store:        store,
		}
	}
}

func (s *instrumentSource) Ack(id string) (err error) {
	defer func(begin time.Time) {
		s.track("Ack", "", begin, err)
	}(time.Now())

	return s.next.Ack(id)
}

func (s *instrumentSource) Consume() (change *StateChange, err error) {
	defer func(begin time.Time) {
		ns := ""

		if err == nil && change != nil {
			ns = change.Namespace

			if !change.SentAt.IsZero() {
				s.queueLatency.With(prometheus.Labels{
					metrics.FieldComponent: s.component,
					metrics.FieldMethod:    "Consume",
					metrics.FieldNamespace: ns,
					metrics.FieldSource:    serviceName,
					metrics.FieldStore:     s.store,
				}).Observe(time.Since(change.SentAt).Seconds())
			}
		}

		s.track("Consume", ns, begin, err)
	}(time.Now())

	return s.next.Consume()
}

func (s *instrumentSource) Propagate(
	ns string,
	old, new *Connection,
) (id string, err error) {
	defer func(begin time.Time) {
		s.track("Propagate", ns, begin, err)
	}(time.Now())

	return s.next.Propagate(ns, old, new)
}

func (s *instrumentSource) track(
	method, namespace string,
	begin time.Time,
	err error,
) {
	if err != nil {
		s.errCount.With(
			metrics.FieldComponent, s.component,
			metrics.FieldMethod, method,
			metrics.FieldNamespace, namespace,
			metrics.FieldSource, serviceName,
			metrics.FieldStore, s.store,
		).Add(1)
	} else {
		s.opCount.With(
			metrics.FieldComponent, s.component,
			metrics.FieldMethod, method,
			metrics.FieldNamespace, namespace,
			metrics.FieldSource, serviceName,
			metrics.FieldStore, s.store,
		).Add(1)

		s.opLatency.With(prometheus.Labels{
			metrics.FieldComponent: s.component,
			metrics.FieldMethod:    method,
			metrics.FieldNamespace: namespace,
			metrics.FieldSource:    serviceName,
			metrics.FieldStore:     s.store,
		}).Observe(time.Since(begin).Seconds())
	}
}
