package object

import "testing"

func TestAttachmentValidate(t *testing.T) {
	for _, a := range []Attachment{
		// Missing Contents
		{
			Name: "attach1",
			Type: AttachmentTypeText,
		},
		// Empty Contents
		{
			Contents: Contents{},
			Name:     "attach",
			Type:     AttachmentTypeText,
		},
		// Missing Name
		{
			Contents: Contents{
				"en": "Lorem ipsum.",
			},
			Name: "",
			Type: AttachmentTypeText,
		},
		// Missing Type
		{
			Contents: Contents{
				"en": "Lorem ipsum.",
			},
			Name: "teaser",
			Type: "",
		},
		// Unspported Type
		{
			Contents: Contents{
				"en": "Lorem ipsum.",
			},
			Name: "teaser",
			Type: "teaser",
		},
		// Invalid language tag
		{
			Contents: Contents{
				"foo-FOO": "Lorem ipsum",
			},
			Name: "body",
			Type: AttachmentTypeText,
		},
		// Invalid URL
		{
			Contents: Contents{
				"en": "http://bit.ly^fake",
			},
			Name: "attach2",
			Type: AttachmentTypeURL,
		},
	} {
		if have, want := a.Validate(), ErrInvalidAttachment; !IsInvalidAttachment(have) {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}

func TestObjectMatchOpts(t *testing.T) {
	var (
		owned = true
		o     = &Object{
			Deleted: false,
			Owned:   false,
			Tags: []string{
				"tag1",
				"tag2",
			},
			Type: "foo",
		}
		cases = map[*QueryOptions]bool{
			nil: true,
			&QueryOptions{Deleted: true}:                  false,
			&QueryOptions{Deleted: false}:                 true,
			&QueryOptions{Owned: &owned}:                  false,
			&QueryOptions{Tags: []string{"tag3", "tag4"}}: false,
			&QueryOptions{Tags: []string{"tag1"}}:         true,
			&QueryOptions{Tags: []string{"tag1", "tag2"}}: true,
			&QueryOptions{Types: []string{"bar"}}:         false,
			&QueryOptions{Types: []string{"foo"}}:         true,
		}
	)

	for opts, want := range cases {
		if have := o.MatchOpts(opts); have != want {
			t.Errorf("have %v, want %v: %v", have, want, opts)
		}
	}
}

func TestObjectValidate(t *testing.T) {
	cases := List{
		// Too many Attachments
		{
			Attachments: []Attachment{
				{},
				{},
				{},
				{},
				{},
				{},
			},
		},
		{
			Type:       "post",
			Visibility: VisibilityConnection,
		},
		// Too many Tags
		{
			OwnerID: 123,
			Tags: []string{
				"tag",
				"tag",
				"tag",
				"tag",
				"tag",
				"tag",
				"tag",
				"tag",
				"tag",
				"tag",
				"tag",
			},
			Type:       "post",
			Visibility: VisibilityConnection,
		},
		// Missing Type
		{
			OwnerID:    123,
			Visibility: VisibilityConnection,
		},
		// Invalid Visibility
		{
			OwnerID:    123,
			Type:       "recipe",
			Visibility: 50,
		},
	}

	for _, o := range cases {
		if have, want := o.Validate(), ErrInvalidObject; !IsInvalidObject(have) {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
