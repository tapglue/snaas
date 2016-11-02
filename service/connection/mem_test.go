package connection

import "testing"

func TestMemCount(t *testing.T) {
	testServiceCount(t, prepareMem)
}

func TestMemPut(t *testing.T) {
	testServicePut(t, prepareMem)
}

func TestMemPutInvalid(t *testing.T) {
	testServicePutInvalid(t, prepareMem)
}

func TestMemQuery(t *testing.T) {
	testServiceQuery(t, prepareMem)
}

func prepareMem(t *testing.T, ns string) Service {
	return MemService()
}
