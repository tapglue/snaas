package reaction

import "testing"

func TestMemCount(t *testing.T) {
	testServiceCount(prepareMem, t)
}

func TestMemCountMulti(t *testing.T) {
	testServiceCountMulti(prepareMem, t)
}

func TestMemPut(t *testing.T) {
	testServicePut(prepareMem, t)
}

func TestMemQuery(t *testing.T) {
	testServiceQuery(prepareMem, t)
}

func prepareMem(t *testing.T, ns string) Service {
	return MemService()
}
