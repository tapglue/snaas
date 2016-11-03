package http

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/tapglue/snaas/service/event"
)

type payloadEvent struct {
	event *event.Event
}

func (p *payloadEvent) MarshalJSON() ([]byte, error) {
	f := struct {
		ID           uint64         `json:"id"`
		IDString     string         `json:"id_string"`
		Language     string         `json:"language"`
		Metadata     event.Metadata `json:"metadata,omitempty"`
		Object       *event.Object  `json:"object"`
		ObjectID     string         `json:"tg_object_id"`
		Owned        bool           `json:"owned"`
		PostID       string         `json:"post_id"`
		Target       *event.Target  `json:"target,omitempty"`
		Type         string         `json:"type"`
		UserID       uint64         `json:"user_id"`
		UserIDString string         `json:"user_id_string"`
		Visibility   uint8          `json:"visibility"`
		CreatedAt    time.Time      `json:"created_at"`
		UpdatedAt    time.Time      `json:"updated_at"`
	}{
		ID:           p.event.ID,
		IDString:     strconv.FormatUint(p.event.ID, 10),
		Language:     p.event.Language,
		Metadata:     p.event.Metadata,
		ObjectID:     strconv.FormatUint(p.event.ObjectID, 10),
		Owned:        p.event.Owned,
		PostID:       strconv.FormatUint(p.event.ObjectID, 10),
		Type:         p.event.Type,
		UserID:       p.event.UserID,
		UserIDString: strconv.FormatUint(p.event.UserID, 10),
		Visibility:   uint8(p.event.Visibility),
		CreatedAt:    p.event.CreatedAt,
		UpdatedAt:    p.event.UpdatedAt,
	}

	if p.event.Object != nil {
		f.Object = p.event.Object
	}

	if p.event.Target != nil {
		f.Target = p.event.Target
	}

	return json.Marshal(f)
}

func (p *payloadEvent) UnmarshalJSON(raw []byte) error {
	f := struct {
		Language   string         `json:"language"`
		Metadata   event.Metadata `json:"metadata,omitempty"`
		Object     *payloadObject `json:"object"`
		Target     *event.Target  `json:"target,omitempty"`
		Type       string         `json:"type"`
		Visibility uint8          `json:"visibility"`
	}{}

	err := json.Unmarshal(raw, &f)
	if err != nil {
		return err
	}

	e := &event.Event{
		Language:   f.Language,
		Metadata:   f.Metadata,
		Target:     f.Target,
		Type:       f.Type,
		Visibility: event.Visibility(f.Visibility),
	}

	if f.Object != nil {
		e.Object = &event.Object{
			DisplayNames: f.Object.DisplayNames,
			ID:           f.Object.ID,
			Type:         f.Object.Type,
			URL:          f.Object.URL,
		}
	}

	p.event = e

	return nil
}

type payloadObject struct {
	DisplayNames map[string]string
	ID           string
	Type         string
	URL          string
}

func (p *payloadObject) UnmarshalJSON(raw []byte) error {
	f := struct {
		DisplayNames map[string]string `json:"display_names,omitempty"`
		ID           interface{}       `json:"id"`
		Type         string            `json:"type"`
		URL          string            `json:"url"`
	}{}

	err := json.Unmarshal(raw, &f)
	if err != nil {
		return err
	}

	p.DisplayNames = f.DisplayNames
	p.Type = f.Type
	p.URL = f.URL

	id, err := parseID(f.ID)
	if err != nil {
		return err
	}

	p.ID = id

	return nil
}

func eventCursorAfter(es event.List, limit int) string {
	var after string

	if len(es) != 0 {
		after = toTimeCursor(es[0].CreatedAt)
	}

	return after
}

func eventCursorBefore(es event.List, limit int) string {
	var before string

	if len(es) != 0 {
		before = toTimeCursor(es[len(es)-1].CreatedAt)
	}

	return before
}

func parseID(input interface{}) (string, error) {
	var id string

	switch t := input.(type) {
	case float64:
		id = fmt.Sprintf("%d", int64(t))
	case int:
		id = strconv.Itoa(t)
	case int64:
		id = strconv.FormatInt(t, 10)
	case uint64:
		id = strconv.FormatUint(t, 10)
	case string:
		id = t
	default:
		return "", fmt.Errorf("unexpected value for id")
	}

	return id, nil
}
