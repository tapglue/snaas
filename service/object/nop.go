package object

type nopSource struct{}

// NopSource returns a noop implementation of Source.
func NopSource() Source {
	return &nopSource{}
}

func (s *nopSource) Ack(id string) error {
	return nil
}

func (s *nopSource) Consume() (*StateChange, error) {
	return &StateChange{}, nil
}

func (s *nopSource) Propagate(ns string, old, new *Object) (string, error) {
	return "", nil
}
