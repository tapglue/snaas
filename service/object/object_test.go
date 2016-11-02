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

func TestObjectValidate(t *testing.T) {
	for _, o := range []*Object{
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
	} {
		if have, want := o.Validate(), ErrInvalidObject; !IsInvalidObject(have) {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
