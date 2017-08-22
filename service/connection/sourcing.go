package connection

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

func (s *sourcingService) Friends(ns string, origin uint64) (List, error) {
	return s.service.Friends(ns, origin)
}

func (s *sourcingService) Put(
	ns string,
	input *Connection,
) (new *Connection, err error) {
	var old *Connection

	defer func() {
		if err == nil {
			_, _ = s.producer.Propagate(ns, old, new)
		}
	}()

	cs, err := s.service.Query(ns, QueryOptions{
		FromIDs: []uint64{
			input.FromID,
		},
		ToIDs: []uint64{
			input.ToID,
		},
		Types: []Type{
			input.Type,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(cs) == 1 {
		old = cs[0]
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
