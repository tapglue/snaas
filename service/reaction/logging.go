package reaction

import (
	"time"

	"github.com/go-kit/kit/log"
)

type logSource struct {
	logger log.Logger
	next   Source
}

// LogSourceMiddleware given a Logger wraps the next Source with logging
// capabilities.
func LogSourceMiddleware(store string, logger log.Logger) SourceMiddleware {
	return func(next Source) Source {
		logger = log.NewContext(logger).With(
			"source", "reaction",
			"store", store,
		)

		return &logSource{
			logger: logger,
			next:   next,
		}
	}
}

func (s *logSource) Ack(id string) (err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"ack_id", id,
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Ack",
		}

		if err != nil {
			ps = append(ps, "err", err)
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Ack(id)
}

func (s *logSource) Consume() (change *StateChange, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Consume",
		}

		if change != nil {
			ps = append(ps,
				"namespace", change.Namespace,
				"reaction_new", change.New,
				"reaction_old", change.Old,
			)
		}

		if err != nil {
			ps = append(ps, "err", err)
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Consume()
}

func (s *logSource) Propagate(
	ns string,
	old, new *Reaction,
) (id string, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"duration_ns", time.Since(begin).Nanoseconds(),
			"id", id,
			"method", "Propagate",
			"namespace", ns,
			"event_new", new,
			"event_old", old,
		}

		if err != nil {
			ps = append(ps, "err", err)
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Propagate(ns, old, new)
}

type logService struct {
	logger log.Logger
	next   Service
}

// LogServiceMiddleware given a logger wraps the next Service with logging
// capabilities.
func LogServiceMiddleware(logger log.Logger, store string) ServiceMiddleware {
	return func(next Service) Service {
		logger = log.NewContext(logger).With(
			"service", "reaction",
			"store", store,
		)

		return &logService{logger: logger, next: next}
	}
}

func (s *logService) Count(ns string, opts QueryOptions) (count uint, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"count", count,
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Count",
			"namespace", ns,
			"opts", opts,
		}

		if err != nil {
			ps = append(ps, "err", err)
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Count(ns, opts)
}

func (s *logService) Put(ns string, input *Reaction) (output *Reaction, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Put",
			"namespace", ns,
			"reaction_input", input,
			"reaction_output", output,
		}

		if err != nil {
			ps = append(ps, "err", err)
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Put(ns, input)
}

func (s *logService) Query(ns string, opts QueryOptions) (list List, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Query",
			"namespace", ns,
			"reaction_len", len(list),
			"opts", opts,
		}

		if err != nil {
			ps = append(ps, "err", err)
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Query(ns, opts)
}

func (s *logService) Setup(ns string) (err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Setup",
			"namespace", ns,
		}

		if err != nil {
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Setup(ns)
}

func (s *logService) Teardown(ns string) (err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Teardown",
			"namespace", ns,
		}

		if err != nil {
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Setup(ns)
}
