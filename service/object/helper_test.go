package object

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

type prepareFunc func(string, *testing.T) Service

var testArticle = &Object{
	OwnerID:    555,
	Type:       "article",
	Visibility: VisibilityGlobal,
}

var testAttachmentText = TextAttachment("intro", Contents{
	"en": "Cupcake ipsum dolor sit amet.",
})

var testAttachmentURL = URLAttachment("teaser", Contents{
	"en": "http://bit.ly/1Jp8bMP",
})

var testInvalid = &Object{
	Attachments: []Attachment{
		{
			Contents: Contents{
				"en": "foo barbaz",
			},
			Name: "summary",
			Type: "invalid",
		},
	},
	Type:       "test",
	Visibility: VisibilityPrivate,
}

var testPost = &Object{
	Attachments: []Attachment{
		testAttachmentText,
		testAttachmentURL,
	},
	OwnerID:    123,
	Tags:       []string{"guide", "diy"},
	Type:       "post",
	Visibility: VisibilityConnection,
}

var testRecipe = &Object{
	Attachments: []Attachment{
		TextAttachment("yum", Contents{
			"en": "Cupcake ipsum dolor sit amet.",
		}),
	},
	OwnerID:    321,
	Tags:       []string{"low-carb", "cold"},
	Type:       "recipe",
	Visibility: VisibilityConnection,
}

func testCreateSet(objectID uint64, start time.Time) []*Object {
	set := []*Object{}

	for i := 0; i < 5; i++ {
		set = append(set, &Object{
			OwnerID:    1,
			Type:       "article",
			Visibility: VisibilityConnection,
		})
	}

	for i := 0; i < 5; i++ {
		set = append(set, &Object{
			OwnerID:    1,
			Type:       "review",
			Visibility: VisibilityPublic,
		})
	}

	for i := 0; i < 5; i++ {
		set = append(set, &Object{
			OwnerID:    2,
			ObjectID:   objectID,
			Type:       "comment",
			Visibility: VisibilityGlobal,
		})
	}

	for i := 0; i < 13; i++ {
		set = append(set, &Object{
			OwnerID:    4,
			ObjectID:   objectID,
			Owned:      true,
			Type:       "tg_comment",
			Visibility: VisibilityConnection,
		})
	}

	for i := 0; i < 7; i++ {
		set = append(set, &Object{
			ExternalID: "external-input-123",
			OwnerID:    5,
			Owned:      true,
			Type:       "tg_comment",
			Visibility: VisibilityConnection,
		})
	}

	for i := 0; i < 3; i++ {
		set = append(set, &Object{
			OwnerID: 6,
			Tags: []string{
				"one",
				"two",
				"three",
			},
			Type:       "tagged",
			Visibility: VisibilityConnection,
		})
	}

	for i := 1; i < 12; i++ {
		set = append(set, &Object{
			OwnerID:    7,
			Type:       "tg_past",
			Visibility: VisibilityPrivate,
			CreatedAt:  start.Add(-time.Duration(time.Duration(i) * time.Hour)),
		})
	}

	return set
}

func testServiceCount(t *testing.T, p prepareFunc) {
	var (
		namespace  = "service_count"
		service    = p(namespace, t)
		testObject = *testArticle

		owned bool
	)

	article, err := service.Put(namespace, &testObject)
	if err != nil {
		t.Fatal(err)
	}

	for _, o := range testCreateSet(article.ID, time.Now()) {
		_, err = service.Put(namespace, o)
		if err != nil {
			t.Fatal(err)
		}
	}

	count, err := service.Count(namespace, QueryOptions{
		OwnerIDs: []uint64{
			1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 10; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		ObjectIDs: []uint64{
			article.ID,
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
		Visibilities: []Visibility{
			VisibilityPublic,
			VisibilityGlobal,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 11; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	owned = true

	count, err = service.Count(namespace, QueryOptions{
		Owned: &owned,
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 20; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		Owned: &owned,
		Types: []string{
			"tg_comment",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 20; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		Tags: []string{
			"one",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 3; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		Tags: []string{
			"one",
			"two",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 3; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		Tags: []string{
			"one",
			"three",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 3; have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	count, err = service.Count(namespace, QueryOptions{
		ExternalIDs: []string{
			"external-input-123",
		},
		Owned: &owned,
		Types: []string{
			"tg_comment",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := count, 7; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testServiceCountMulti(t *testing.T, p prepareFunc) {
	var (
		namespace = "service_count_multi"
		service   = p(namespace, t)
		objectIDs = []uint64{
			uint64(rand.Int63()),
			uint64(rand.Int63()),
			uint64(rand.Int63()),
		}
		ownerID = uint64(rand.Int63())
		want    = CountsMap{}
	)

	for _, oid := range objectIDs {
		article := *testArticle

		article.ObjectID = oid
		article.OwnerID = ownerID

		_, err := service.Put(namespace, &article)
		if err != nil {
			t.Fatal(err)
		}

		it := rand.Intn(12)

		for i := 0; i < it; i++ {
			_, err = service.Put(namespace, &Object{
				ObjectID:   oid,
				Owned:      true,
				OwnerID:    uint64(rand.Int63()),
				Type:       TypeComment,
				Visibility: VisibilityPublic,
			})
			if err != nil {
				t.Fatal(err)
			}
		}

		want[oid] = Counts{
			Comments: uint64(it),
		}
	}

	have, err := service.CountMulti(namespace, objectIDs...)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func testServiceQuery(t *testing.T, p prepareFunc) {
	var (
		namespace  = "service_query"
		service    = p(namespace, t)
		testObject = *testArticle
		owned      = true
		notOwned   = false
		start      = time.Now()
	)

	article, err := service.Put(namespace, &testObject)
	if err != nil {
		t.Fatal(err)
	}

	for _, o := range testCreateSet(article.ID, start) {
		_, err = service.Put(namespace, o)
		if err != nil {
			t.Fatal(err)
		}
	}

	cases := map[*QueryOptions]int{
		&QueryOptions{Before: start.Add(-(time.Hour + time.Minute))}:                                             10,
		&QueryOptions{Limit: 5}:                                                                                  5,
		&QueryOptions{ObjectIDs: []uint64{article.ID}, Owned: &notOwned}:                                         5,
		&QueryOptions{Owned: &owned}:                                                                             20,
		&QueryOptions{Owned: &owned, Types: []string{"tg_comment"}}:                                              20,
		&QueryOptions{OwnerIDs: []uint64{1}}:                                                                     10,
		&QueryOptions{Tags: []string{"one"}}:                                                                     3,
		&QueryOptions{Tags: []string{"one", "two"}}:                                                              3,
		&QueryOptions{Tags: []string{"one", "three"}}:                                                            3,
		&QueryOptions{Visibilities: []Visibility{VisibilityPublic, VisibilityGlobal}}:                            11,
		&QueryOptions{ExternalIDs: []string{"external-input-123"}, Owned: &owned, Types: []string{"tg_comment"}}: 7,
	}

	for opts, want := range cases {
		os, err := service.Query(namespace, *opts)
		if err != nil {
			t.Fatal(err)
		}

		if have := len(os); have != want {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
