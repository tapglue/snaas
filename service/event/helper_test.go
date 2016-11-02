package event

import (
	"reflect"
	"testing"
	"time"
)

type prepareFunc func(namespace string, t *testing.T) Service

func testServiceCount(p prepareFunc, t *testing.T) {
	var (
		namespace         = "service_count"
		service           = p(namespace, t)
		enabled           = true
		externalID        = "external-id-123"
		objectID   uint64 = 321
		owned             = false
		targetID          = "123"
	)

	for _, e := range testList(objectID, externalID, targetID, time.Now()) {
		_, err := service.Put(namespace, e)
		if err != nil {
			t.Fatal(err)
		}
	}

	count, err := service.Count(namespace, QueryOptions{
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 56; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		ObjectIDs: []uint64{
			objectID,
		},
		Owned: &owned,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 5; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		UserIDs: []uint64{
			1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 5; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		Visibilities: []Visibility{
			VisibilityPublic,
			VisibilityGlobal,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 10; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	owned = true

	count, err = service.Count(namespace, QueryOptions{
		Owned: &owned,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 11; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		Owned: &owned,
		Types: []string{
			"tg_like",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 6; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		ExternalObjectIDs: []string{
			externalID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 11; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		ExternalObjectTypes: []string{
			"restaurant",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 11; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		TargetIDs: []string{
			targetID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 3; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		TargetTypes: []string{
			TargetUser,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 3; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testEvent() *Event {
	return &Event{
		Enabled:    true,
		Type:       "rate",
		UserID:     1,
		Visibility: VisibilityConnection,
	}
}

func testList(objectID uint64, externalID, targetID string, start time.Time) List {
	es := List{}

	for i := 0; i < 5; i++ {
		es = append(es, &Event{
			Enabled:    true,
			Type:       "bookmark",
			UserID:     1,
			Visibility: VisibilityConnection,
		})
	}

	for i := 0; i < 5; i++ {
		es = append(es, &Event{
			Enabled:    true,
			ObjectID:   objectID,
			Type:       "share",
			UserID:     2,
			Visibility: VisibilityConnection,
		})
	}

	for i := 0; i < 5; i++ {
		es = append(es, &Event{
			Enabled:    true,
			Type:       "play",
			UserID:     2,
			Visibility: VisibilityPublic,
		})
	}

	for i := 0; i < 5; i++ {
		es = append(es, &Event{
			Enabled:    true,
			Type:       "vote",
			UserID:     2,
			Visibility: VisibilityPublic,
		})
	}

	for i := 0; i < 5; i++ {
		es = append(es, &Event{
			Enabled:    true,
			ObjectID:   objectID,
			Owned:      true,
			Type:       "tg_share",
			UserID:     4,
			Visibility: VisibilityConnection,
		})
	}

	for i := 0; i < 6; i++ {
		es = append(es, &Event{
			Enabled:    true,
			ObjectID:   objectID,
			Owned:      true,
			Type:       "tg_like",
			UserID:     4,
			Visibility: VisibilityConnection,
		})
	}

	for i := 0; i < 11; i++ {
		es = append(es, &Event{
			Enabled: true,
			Object: &Object{
				ID:   externalID,
				Type: "restaurant",
			},
			Type:       "checkin",
			UserID:     4,
			Visibility: VisibilityConnection,
		})
	}

	for i := 0; i < 3; i++ {
		es = append(es, &Event{
			Enabled: true,
			Target: &Target{
				ID:   targetID,
				Type: TargetUser,
			},
			Type:       "taggin",
			UserID:     5,
			Visibility: VisibilityPrivate,
		})
	}

	for i := 1; i < 12; i++ {
		es = append(es, &Event{
			CreatedAt:  start.Add(-(time.Duration(i) * time.Hour)),
			Enabled:    true,
			Type:       "tg_past",
			UserID:     7,
			Visibility: VisibilityConnection,
		})
	}

	return es
}

func testServicePut(p prepareFunc, t *testing.T) {
	var (
		event     = testEvent()
		namespace = "service_put"
		service   = p(namespace, t)
		enabled   = true
	)

	created, err := service.Put(namespace, event)
	if err != nil {
		t.Fatal(err)
	}

	list, err := service.Query(namespace, QueryOptions{
		Enabled: &enabled,
		IDs: []uint64{
			created.ID,
		},
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

	created.Enabled = false

	updated, err := service.Put(namespace, created)
	if err != nil {
		t.Fatal(err)
	}

	list, err = service.Query(namespace, QueryOptions{
		Enabled: &created.Enabled,
		IDs: []uint64{
			created.ID,
		},
	})

	if have, want := len(list), 1; have != want {
		t.Fatalf("have %v, want %v", have, want)
	}
	if have, want := list[0], updated; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testServiceQuery(p prepareFunc, t *testing.T) {
	var (
		namespace         = "service_query"
		service           = p(namespace, t)
		enabled           = true
		externalID        = "external-id-123"
		objectID   uint64 = 321
		notOwned          = false
		owned             = true
		start             = time.Now()
		targetID          = "432"
	)

	for _, e := range testList(objectID, externalID, targetID, start) {
		_, err := service.Put(namespace, e)
		if err != nil {
			t.Fatal(err)
		}
	}

	cases := map[*QueryOptions]int{
		&QueryOptions{Before: start.Add(-(time.Hour + time.Minute))}:                  10,
		&QueryOptions{Enabled: &enabled}:                                              56,
		&QueryOptions{ExternalObjectIDs: []string{externalID}}:                        11,
		&QueryOptions{ExternalObjectTypes: []string{"restaurant"}}:                    11,
		&QueryOptions{Limit: 9}:                                                       9,
		&QueryOptions{ObjectIDs: []uint64{objectID}, Owned: &notOwned}:                5,
		&QueryOptions{Owned: &owned}:                                                  11,
		&QueryOptions{Owned: &owned, Types: []string{"tg_like"}}:                      6,
		&QueryOptions{TargetIDs: []string{targetID}}:                                  3,
		&QueryOptions{TargetTypes: []string{TargetUser}}:                              3,
		&QueryOptions{UserIDs: []uint64{1}}:                                           5,
		&QueryOptions{Visibilities: []Visibility{VisibilityPublic, VisibilityGlobal}}: 10,
	}

	for opts, want := range cases {
		es, err := service.Query(namespace, *opts)
		if err != nil {
			t.Fatal(err)
		}

		if have := len(es); have != want {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
