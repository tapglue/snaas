package core

import (
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/rule"
)

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
