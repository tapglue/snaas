package event

import "testing"

func TestMemCount(t *testing.T) {
	testServiceCount(prepareMem, t)
}

func TestMemPut(t *testing.T) {
	testServicePut(prepareMem, t)
}

func TestMemQuery(t *testing.T) {
	testServiceQuery(prepareMem, t)
}

func prepareMem(ns string, t *testing.T) Service {
	return MemService()
}
