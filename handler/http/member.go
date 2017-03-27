package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

// MemberLogin authenticates the member via OAuth.
func MemberLogin(authConf *oauth2.Config) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			p = payloadMemberLogin{}
		)

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		t, err := authConf.Exchange(ctx, p.Code)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		c := authConf.Client(ctx, t)

		res, err := c.Get("https://www.googleapis.com/oauth2/v3/userinfo")
		if err != nil {
			respondError(w, 0, err)
			return
		}

		m := struct {
			GivenName string `json:"given_name"`
			Picture   string `json:"picture"`
		}{}

		err = json.NewDecoder(res.Body).Decode(&m)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadMember{
			Auth:    t,
			Name:    m.GivenName,
			Picture: m.Picture,
		})
	}
}

// MemberRetrieveMe returns the current member.
func MemberRetrieveMe(authConf *oauth2.Config) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var (
			t = &oauth2.Token{}
		)

		err := json.NewDecoder(r.Body).Decode(t)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		c := authConf.Client(ctx, t)

		res, err := c.Get("https://www.googleapis.com/oauth2/v3/userinfo")
		if err != nil {
			respondError(w, 0, err)
			return
		}

		m := struct {
			GivenName string `json:"given_name"`
			Picture   string `json:"picture"`
		}{}

		if res.StatusCode >= 400 && res.StatusCode < 500 {
			respondError(w, 0, wrapError(ErrUnauthorized, "invalid token"))
			return
		}

		fmt.Printf("%#v\n", res)

		err = json.NewDecoder(res.Body).Decode(&m)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadMember{
			Auth:    t,
			Name:    m.GivenName,
			Picture: m.Picture,
		})
	}
}

type payloadMember struct {
	Auth    *oauth2.Token `json:"auth"`
	Name    string        `json:"name"`
	Picture string        `json:"picture"`
}

type payloadMemberLogin struct {
	Code string `json:"code"`
}
