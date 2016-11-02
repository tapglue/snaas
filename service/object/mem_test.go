package object

import (
	"reflect"
	"testing"
)

func TestMemServiceCount(t *testing.T) {
	testServiceCount(t, prepareMem)
}

func TestMemServicePut(t *testing.T) {
	var (
		namespace = "service_put"
		service   = prepareMem(namespace, t)
		post      = *testPost
	)

	created, err := service.Put(namespace, &post)
	if err != nil {
		t.Fatal(err)
	}

	list, err := service.Query(namespace, QueryOptions{
		ID: &created.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(list), 1; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}
	if have, want := list[0], created; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}

	created.Deleted = true

	updated, err := service.Put(namespace, created)
	if err != nil {
		t.Fatal(err)
	}

	list, err = service.Query(namespace, QueryOptions{
		Deleted: true,
		ID:      &created.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(list), 1; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}
	if have, want := list[0], updated; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestMemServicePutInvalid(t *testing.T) {
	var (
		namespace = "service_put_invalid"
		service   = prepareMem(namespace, t)
		invalid   = *testInvalid
	)

	_, err := service.Put(namespace, &invalid)
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestMemServiceQuery(t *testing.T) {
	testServiceQuery(t, prepareMem)
}

func prepareMem(namespace string, t *testing.T) Service {
	return MemService()
}
