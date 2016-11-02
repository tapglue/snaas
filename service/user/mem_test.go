package user

import "testing"

func TestMemCount(t *testing.T) {
	testServiceCount(t, prepareMem)
}

func TestMemPut(t *testing.T) {
	testServicePut(t, prepareMem)
}

func TestMemPutLastRead(t *testing.T) {
	testServicePutLastRead(t, prepareMem)
}

func TestMemSearch(t *testing.T) {
	testServiceSearch(t, prepareMem)
}

func prepareMem(t *testing.T, ns string) Service {
	return MemService()
}
