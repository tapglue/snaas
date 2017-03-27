package core

import (
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/rule"
)

// RuleActivateFunc puts the rule in an active state.
type RuleActivateFunc func(appID, id uint64) error

func RuleActivate(
	apps app.Service,
	rules rule.Service,
) RuleActivateFunc {
	return func(appID, id uint64) error {
		currentApp, err := AppFetch(apps)(appID)
		if err != nil {
			return err
		}

		r, err := RuleFetch(apps, rules)(appID, id)
		if err != nil {
			return err
		}

		if r.Active {
			return nil
		}

		r.Active = true

		_, err = rules.Put(currentApp.Namespace(), r)
		return err
	}
}

// RuleDeactivateFunc puts the rule in an inactive state.
type RuleDeactivateFunc func(appID, id uint64) error

// RuleDeactivate puts the rule in an inactive state.
func RuleDeactivate(
	apps app.Service,
	rules rule.Service,
) RuleDeactivateFunc {
	return func(appID, id uint64) error {
		currentApp, err := AppFetch(apps)(appID)
		if err != nil {
			return err
		}

		r, err := RuleFetch(apps, rules)(appID, id)
		if err != nil {
			return err
		}

		if !r.Active {
			return nil
		}

		r.Active = false

		_, err = rules.Put(currentApp.Namespace(), r)
		return err
	}
}

// RuleDeleteFunc removes the rule permanently.
type RuleDeleteFunc func(appID, id uint64) error

// RuleDelete removes the rule permanently.
func RuleDelete(
	apps app.Service,
	rules rule.Service,
) RuleDeleteFunc {
	return func(appID, id uint64) error {
		currentApp, err := AppFetch(apps)(appID)
		if err != nil {
			return err
		}

		r, err := RuleFetch(apps, rules)(appID, id)
		if err != nil {
			return err
		}

		r.Deleted = true

		_, err = rules.Put(currentApp.Namespace(), r)
		if err != nil {
			return err
		}

		return nil
	}
}

// RuleFetchFunc returns the Rule for the given appID and id.
type RuleFetchFunc func(appID, id uint64) (*rule.Rule, error)

// RuleFetch returns the Rule for the given appID and id.
func RuleFetch(apps app.Service, rules rule.Service) RuleFetchFunc {
	return func(appID, id uint64) (*rule.Rule, error) {
		currentApp, err := AppFetch(apps)(appID)
		if err != nil {
			return nil, err
		}

		rs, err := rules.Query(currentApp.Namespace(), rule.QueryOptions{
			IDs: []uint64{
				id,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(rs) == 0 {
			return nil, wrapError(ErrNotFound, "rule (%d) not found", id)
		}

		return rs[0], nil
	}
}

// RuleListFunc returns all rules for the current App.
type RuleListFunc func(appID uint64) (rule.List, error)

// RuleList returns all rules for the current App.
func RuleList(apps app.Service, rules rule.Service) RuleListFunc {
	return func(appID uint64) (rule.List, error) {
		currentApp, err := AppFetch(apps)(appID)
		if err != nil {
			return nil, err
		}

		return rules.Query(currentApp.Namespace(), rule.QueryOptions{
			Deleted: &defaultDeleted,
		})
	}
}

// RuleListActiveFunc returns all active rules for the current App.
type RuleListActiveFunc func(*app.App, rule.Type) (rule.List, error)

// RuleListActive returns all active rules for the current App.
func RuleListActive(rules rule.Service) RuleListActiveFunc {
	return func(currentApp *app.App, ruleType rule.Type) (rule.List, error) {
		return rules.Query(currentApp.Namespace(), rule.QueryOptions{
			Active:  &defaultActive,
			Deleted: &defaultDeleted,
			Types: []rule.Type{
				ruleType,
			},
		})
	}
}
