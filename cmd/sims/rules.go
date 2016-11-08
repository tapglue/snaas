package main

import (
	"fmt"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/user"
)

// Messaging.
const (
	fmtCommentPost     = `%s commented on a Post.`
	fmtCommentPostOwn  = `%s commented on your Post.`
	fmtFollow          = `%s started following you`
	fmtFriendConfirmed = `%s accepted your friend request.`
	fmtFriendRequest   = `%s sent you a friend request.`
	fmtLikePost        = `%s liked a Post.`
	fmtLikePostOwn     = `%s liked your Post.`
	fmtPostCreated     = `%s created a new Post.`
)

// Deep-linking URNs.
const (
	urnComment = `tapglue/posts/%d/comments/%d`
	urnPost    = `tapglue/posts/%d`
	urnUser    = `tapglue/users/%d`
)

type conRuleFunc func(*app.App, *connection.StateChange) (messages, error)
type eventRuleFunc func(*app.App, *event.StateChange) (messages, error)
type objectRuleFunc func(*app.App, *object.StateChange) (messages, error)

func conRuleFollower(userFetch core.UserFetchFunc) conRuleFunc {
	return func(
		currentApp *app.App,
		change *connection.StateChange,
	) (messages, error) {
		if change.Old != nil ||
			change.New.State != connection.StateConfirmed ||
			change.New.Type != connection.TypeFollow {
			return nil, nil
		}

		origin, err := userFetch(currentApp, change.New.FromID)
		if err != nil {
			return nil, fmt.Errorf("origin fetch: %s", err)
		}

		target, err := userFetch(currentApp, change.New.ToID)
		if err != nil {
			return nil, fmt.Errorf("target fetch: %s", err)
		}

		return messages{
			{
				message:   fmtToMessage(fmtFollow, origin),
				recipient: target.ID,
				urn:       fmt.Sprintf(urnUser, origin.ID),
			},
		}, nil
	}
}

func conRuleFriendConfirmed(userFetch core.UserFetchFunc) conRuleFunc {
	return func(
		currentApp *app.App,
		change *connection.StateChange,
	) (messages, error) {
		if change.Old == nil ||
			change.Old.Type != connection.TypeFriend ||
			change.New.State != connection.StatePending ||
			change.New.Type != connection.TypeFriend {
			return nil, nil
		}

		origin, err := userFetch(currentApp, change.New.FromID)
		if err != nil {
			return nil, fmt.Errorf("origin fetch: %s", err)
		}

		target, err := userFetch(currentApp, change.New.ToID)
		if err != nil {
			return nil, fmt.Errorf("target fetch: %s", err)
		}

		return messages{
			{
				message:   fmtToMessage(fmtFriendConfirmed, target),
				recipient: origin.ID,
				urn:       fmt.Sprintf(urnUser, origin.ID),
			},
		}, nil
	}
}

func conRuleFriendRequest(userFetch core.UserFetchFunc) conRuleFunc {
	return func(
		currentApp *app.App,
		change *connection.StateChange,
	) (messages, error) {
		if change.Old != nil ||
			change.New.State != connection.StatePending ||
			change.New.Type != connection.TypeFriend {
			return nil, nil
		}

		origin, err := userFetch(currentApp, change.New.FromID)
		if err != nil {
			return nil, fmt.Errorf("origin fetch: %s", err)
		}

		target, err := userFetch(currentApp, change.New.ToID)
		if err != nil {
			return nil, fmt.Errorf("target fetch: %s", err)
		}

		return messages{
			{
				message:   fmtToMessage(fmtFriendRequest, origin),
				recipient: target.ID,
				urn:       fmt.Sprintf(urnUser, origin.ID),
			},
		}, nil
	}
}

func eventRuleLikeCreated(
	followerIDs core.ConnectionFollowerIDsFunc,
	friendIDs core.ConnectionFriendIDsFunc,
	postFetch core.PostFetchFunc,
	userFetch core.UserFetchFunc,
	usersFetch core.UsersFetchFunc,
) eventRuleFunc {
	return func(
		currentApp *app.App,
		change *event.StateChange,
	) (messages, error) {
		if change.Old != nil ||
			change.New.Enabled == false ||
			!core.IsLike(change.New) {
			return nil, nil
		}

		post, err := postFetch(currentApp, change.New.ObjectID)
		if err != nil {
			return nil, fmt.Errorf("post fetch: %s", err)
		}

		origin, err := userFetch(currentApp, change.New.UserID)
		if err != nil {
			return nil, fmt.Errorf("origin fetch: %s", err)
		}

		owner, err := userFetch(currentApp, post.OwnerID)
		if err != nil {
			return nil, fmt.Errorf("owner fetch: %s", err)
		}

		followIDs, err := followerIDs(currentApp, origin.ID)
		if err != nil {
			return nil, err
		}

		fIDs, err := friendIDs(currentApp, origin.ID)
		if err != nil {
			return nil, err
		}

		ids := filterIDs(append(followIDs, fIDs...), owner.ID)

		rs, err := usersFetch(currentApp, ids...)
		if err != nil {
			return nil, err
		}

		rs = append(rs, owner)
		ms := messages{}

		for _, recipient := range rs {
			f := fmtLikePost

			if post.OwnerID == recipient.ID {
				f = fmtLikePostOwn
			}

			ms = append(ms, &message{
				message:   fmtToMessage(f, origin),
				recipient: recipient.ID,
				urn:       fmt.Sprintf(urnPost, post.ID),
			})
		}

		return ms, nil
	}
}

func objectRuleCommentCreated(
	followerIDs core.ConnectionFollowerIDsFunc,
	friendIDs core.ConnectionFriendIDsFunc,
	postFetch core.PostFetchFunc,
	userFetch core.UserFetchFunc,
	usersFetch core.UsersFetchFunc,
) objectRuleFunc {
	return func(
		currentApp *app.App,
		change *object.StateChange,
	) (messages, error) {
		if change.Old != nil ||
			change.New.Deleted == true ||
			!core.IsComment(change.New) {
			return nil, nil
		}

		post, err := postFetch(currentApp, change.New.ObjectID)
		if err != nil {
			return nil, fmt.Errorf("post fetch: %s", err)
		}

		origin, err := userFetch(currentApp, change.New.OwnerID)
		if err != nil {
			return nil, fmt.Errorf("origin fetch: %s", err)
		}

		owner, err := userFetch(currentApp, post.OwnerID)
		if err != nil {
			return nil, fmt.Errorf("owner fetch: %s", err)
		}

		followIDs, err := followerIDs(currentApp, origin.ID)
		if err != nil {
			return nil, err
		}

		fIDs, err := friendIDs(currentApp, origin.ID)
		if err != nil {
			return nil, err
		}

		ids := filterIDs(append(followIDs, fIDs...), owner.ID)

		rs, err := usersFetch(currentApp, ids...)
		if err != nil {
			return nil, err
		}

		rs = append(rs, owner)
		ms := messages{}

		for _, recipient := range rs {
			f := fmtCommentPost

			if post.OwnerID == recipient.ID {
				f = fmtCommentPostOwn
			}

			ms = append(ms, &message{
				message:   fmtToMessage(f, origin),
				recipient: recipient.ID,
				urn:       fmt.Sprintf(urnComment, post.ID, change.New.ID),
			})
		}

		return ms, nil
	}
}

func objectRulePostCreated(
	followerIDs core.ConnectionFollowerIDsFunc,
	friendIDs core.ConnectionFriendIDsFunc,
	userFetch core.UserFetchFunc,
	usersFetch core.UsersFetchFunc,
) objectRuleFunc {
	return func(
		currentApp *app.App,
		change *object.StateChange,
	) (messages, error) {
		if change.Old != nil ||
			change.New.Deleted == true ||
			!core.IsPost(change.New) {
			return nil, nil
		}

		origin, err := userFetch(currentApp, change.New.OwnerID)
		if err != nil {
			return nil, fmt.Errorf("origin fetch: %s", err)
		}

		followIDs, err := followerIDs(currentApp, origin.ID)
		if err != nil {
			return nil, err
		}

		fIDs, err := friendIDs(currentApp, origin.ID)
		if err != nil {
			return nil, err
		}

		rs, err := usersFetch(currentApp, append(followIDs, fIDs...)...)
		if err != nil {
			return nil, err
		}

		ms := messages{}

		for _, recipient := range rs {
			ms = append(ms, &message{
				message:   fmtToMessage(fmtPostCreated, origin),
				recipient: recipient.ID,
				urn:       fmt.Sprintf(urnPost, change.New.ID),
			})
		}

		return ms, nil
	}
}

func filterIDs(ids []uint64, fs ...uint64) []uint64 {
	var (
		is   = []uint64{}
		seen = map[uint64]struct{}{}
	)

	for _, id := range fs {
		seen[id] = struct{}{}
	}

	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}

		is = append(is, id)
	}

	return is
}

func fmtToMessage(f string, u *user.User) string {
	if u.Firstname != "" {
		return fmt.Sprintf(f, u.Firstname)
	}

	return fmt.Sprintf(f, u.Username)
}
