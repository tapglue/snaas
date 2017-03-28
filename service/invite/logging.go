package invite

import (
	"time"

	"github.com/go-kit/kit/log"
)

type logService struct {
	logger log.Logger
	next   Service
}

// LogServiceMiddleware given a logger wraps the next Service with logging
// capabilities.
func LogServiceMiddleware(logger log.Logger, store string) ServiceMiddleware {
	return func(next Service) Service {
		logger = log.With(
			logger,
			"service", entity,
			"store", store,
		)

		return &logService{logger: logger, next: next}
	}
}

func (s *logService) Put(ns string, input *Invite) (output *Invite, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"duration_ns", time.Since(begin).Nanoseconds(),
			"method", "Put",
			"namespace", ns,
			"invite_input", input,
			"invite_output", output,
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
			"invite_len", len(list),
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
