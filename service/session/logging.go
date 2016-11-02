package session

import (
	"time"

	"github.com/go-kit/kit/log"
)

type logService struct {
	logger log.Logger
	next   Service
}

// LogMiddleware given a Logger wraps the next Service with logging capabilities.
func LogMiddleware(logger log.Logger, store string) ServiceMiddleware {
	return func(next Service) Service {
		logger = log.NewContext(logger).With(
			"service", "session",
			"store", store,
		)

		return &logService{logger: logger, next: next}
	}
}

func (s *logService) Put(
	ns string,
	input *Session,
) (output *Session, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"duration_ns", time.Since(begin).Nanoseconds(),
			"session_input", input,
			"method", "Put",
			"namespace", ns,
			"session_output", output,
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
			"session_len", len(list),
			"session_opts", opts,
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
