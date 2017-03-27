package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"golang.org/x/net/context"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/service/rule"
)

func RuleActivate(fn core.RuleActivateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		appID, err := extractAppID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		ruleID, err := extractRuleID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = fn(appID, ruleID)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

func RuleDeactivate(fn core.RuleDeactivateFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		appID, err := extractAppID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		ruleID, err := extractRuleID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = fn(appID, ruleID)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

func RuleDelete(fn core.RuleDeleteFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		appID, err := extractAppID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		ruleID, err := extractRuleID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		err = fn(appID, ruleID)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}

// RuleList returns all rules.
func RuleList(fn core.RuleListFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		appID, err := extractAppID(r)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		rs, err := fn(appID)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		if len(rs) == 0 {
			respondJSON(w, http.StatusNoContent, nil)
		}

		respondJSON(w, http.StatusOK, &payloadRules{rules: rs})
	}
}

// RuleRetrieve returns a single rule by id.
func RuleRetrieve(fn core.RuleFetchFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, req *http.Request) {
		appID, err := extractAppID(req)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		ruleID, err := extractRuleID(req)
		if err != nil {
			respondError(w, 0, wrapError(ErrBadRequest, err.Error()))
			return
		}

		r, err := fn(appID, ruleID)
		if err != nil {
			respondError(w, 0, err)
			return
		}

		respondJSON(w, http.StatusOK, &payloadRule{rule: r})
	}
}

type payloadRule struct {
	rule *rule.Rule
}

func (p *payloadRule) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Active     bool            `json:"active"`
		Criteria   interface{}     `json:"criteria"`
		Deleted    bool            `json:"deleted"`
		Ecosystem  int             `json:"ecosystem"`
		Entity     int             `json:"entity"`
		ID         string          `json:"id"`
		Name       string          `json:"name"`
		Recipients rule.Recipients `json:"recipients"`
	}{
		Active:     p.rule.Active,
		Criteria:   p.rule.Criteria,
		Deleted:    p.rule.Deleted,
		Ecosystem:  int(p.rule.Ecosystem),
		Entity:     int(p.rule.Type),
		ID:         strconv.FormatUint(p.rule.ID, 10),
		Name:       p.rule.Name,
		Recipients: p.rule.Recipients,
	})
}

type payloadRules struct {
	rules rule.List
}

func (p *payloadRules) MarshalJSON() ([]byte, error) {
	rs := []*payloadRule{}

	for _, r := range p.rules {
		rs = append(rs, &payloadRule{rule: r})
	}

	return json.Marshal(struct {
		Rules []*payloadRule `json:"rules"`
	}{
		Rules: rs,
	})
}
