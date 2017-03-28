package core

import (
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/invite"
)

// InviteCreateFunc stores the key and value for the users invite.
type InviteCreateFunc func(
	currentApp *app.App,
	origin Origin,
	key, value string,
) error

func InviteCreate(invites invite.Service) InviteCreateFunc {
	return func(
		currentApp *app.App,
		origin Origin,
		key, value string,
	) error {
		is, err := invites.Query(currentApp.Namespace(), invite.QueryOptions{
			Keys: []string{
				key,
			},
			UserIDs: []uint64{
				origin.UserID,
			},
			Values: []string{
				value,
			},
		})
		if err != nil {
			return err
		}

		if len(is) == 1 {
			return nil
		}

		_, err = invites.Put(currentApp.Namespace(), &invite.Invite{
			Key:    key,
			UserID: origin.UserID,
			Value:  value,
		})
		if err != nil {
			return err
		}

		return nil
	}
}
