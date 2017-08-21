package reaction

type sourcingService struct {
	producer Producer
	service  Service
}

func SourcingServiceMiddleware(producer Producer) ServiceMiddleware {
	return func(service Service) Service {
		return &sourcingService{
			service:  service,
			producer: producer,
		}
	}
}

func (s *sourcingService) Count(ns string, opts QueryOptions) (uint, error) {
	return s.service.Count(ns, opts)
}

func (s *sourcingService) CountMulti(ns string, opts QueryOptions) (CountsMap, error) {
	return s.service.CountMulti(ns, opts)
}

func (s *sourcingService) Put(ns string, input *Reaction) (new *Reaction, err error) {
	var old *Reaction

	defer func() {
		if err == nil {
			_, _ = s.producer.Propagate(ns, old, new)
		}
	}()

	if input.ID != 0 {
		rs, err := s.service.Query(ns, QueryOptions{
			IDs: []uint64{
				input.ID,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(rs) == 1 {
			old = rs[0]
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
