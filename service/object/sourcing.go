package object

type sourcingService struct {
	producer Producer
	service  Service
}

// SourcingServiceMiddleware propagates state changes for the Service via the
// given Producer.
func SourcingServiceMiddleware(producer Producer) ServiceMiddleware {
	return func(service Service) Service {
		return &sourcingService{
			producer: producer,
			service:  service,
		}
	}
}

func (s *sourcingService) Count(ns string, opts QueryOptions) (int, error) {
	return s.service.Count(ns, opts)
}

func (s *sourcingService) CountMulti(ns string, objectIDs ...uint64) (m CountsMap, err error) {
	return s.service.CountMulti(ns, objectIDs...)
}

func (s *sourcingService) Put(
	ns string,
	input *Object,
) (new *Object, err error) {
	var old *Object

	defer func() {
		if err == nil {
			_, _ = s.producer.Propagate(ns, old, new)
		}
	}()

	if input.ID != 0 {
		os, err := s.service.Query(ns, QueryOptions{
			ID: &input.ID,
		})
		if err != nil {
			return nil, err
		}

		if len(os) == 1 {
			old = os[0]
		}
	}

	return s.service.Put(ns, input)
}

func (s *sourcingService) Query(ns string, opts QueryOptions) (List, error) {
	return s.service.Query(ns, opts)
}

func (s *sourcingService) Setup(ns string) error {
	return s.service.Setup(ns)
}

func (s *sourcingService) Teardown(ns string) error {
	return s.service.Teardown(ns)
}
