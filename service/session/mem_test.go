package session

import "testing"

func TestMemPut(t *testing.T) {
	testServicePut(t, prepareMem)
}

func TestMemQuery(t *testing.T) {
	testServiecQuery(t, prepareMem)
}

func prepareMem(t *testing.T, ns string) Service {
	return MemService()
}
