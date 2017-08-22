package connection

import (
	"time"

	"github.com/go-kit/kit/log"
)

type logService struct {
	logger log.Logger
	next   Service
}

// LogServiceMiddleware given a Logger wraps the next Service with logging capabilities.
func LogServiceMiddleware(logger log.Logger, store string) ServiceMiddleware {
	return func(next Service) Service {
		logger = log.With(
			logger,
			"service", "connection",
			"store", store,
		)

		return &logService{logger: logger, next: next}
	}
}

func (s *logService) Count(ns string, opts QueryOptions) (count int, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"connection_count", count,
			"connection_opts", opts,
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Count",
			"namespace", ns,
		}

		if err != nil {
			ps = append(ps, "err", err)
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Count(ns, opts)
}

func (s *logService) Friends(ns string, origin uint64) (list List, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"conenction_len", len(list),
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Friends",
			"namespace", ns,
			"origin", origin,
		}

		if err != nil {
			ps = append(ps, "err", err)
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Friends(ns, origin)
}

func (s *logService) Put(
	ns string,
	input *Connection,
) (output *Connection, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"connection_input", input,
			"connection_output", output,
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Put",
			"namespace", ns,
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
			"conenction_len", len(list),
			"connection_opts", opts,
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Query",
			"namespace", ns,
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
			ps = append(ps, "err", err)
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
			ps = append(ps, "err", err)
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Teardown(ns)
}

type logSource struct {
	logger log.Logger
	next   Source
}

// LogSourceMiddleware given a Logger wraps the next Source with logging capabilities.
func LogSourceMiddleware(store string, logger log.Logger) SourceMiddleware {
	return func(next Source) Source {
		logger = log.With(
			logger,
			"source", "connection",
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
				"connection_new", change.New,
				"connection_old", change.Old,
			)
		}

		if err != nil {
			ps = append(ps, "err", err)
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Consume()
}

func (s *logSource) Propagate(ns string, old, new *Connection) (id string, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"connection_new", new,
			"connection_old", old,
			"duration_ns", time.Since(begin).Nanoseconds(),
			"id", id,
			"method", "Propagate",
			"namespace", ns,
		}

		if err != nil {
			ps = append(ps, "err", err)
		}

		_ = s.logger.Log(ps...)
	}(time.Now())

	return s.next.Propagate(ns, old, new)
}
