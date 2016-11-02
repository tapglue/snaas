package device

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
		logger = log.NewContext(logger).With(
			"service", "device",
			"store", store,
		)

		return &logService{logger: logger, next: next}
	}
}

func (s *logService) Put(ns string, input *Device) (output *Device, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"device_input", input,
			"device_output", output,
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

func (s *logService) Query(ns string, opts QueryOptions) (ds List, err error) {
	defer func(begin time.Time) {
		ps := []interface{}{
			"device_len", len(ds),
			"device_opts", opts,
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
