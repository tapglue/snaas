package source

// Acker permantly removes the workload from the Source.
type Acker interface {
	Ack(id string) error
}
