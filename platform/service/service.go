package service

// Lifecycle encodes the functionality necessary to control the full lifecycle
// of a data service.
type Lifecycle interface {
	Setup(namesapce string) error
	Teardown(namespace string) error
}
