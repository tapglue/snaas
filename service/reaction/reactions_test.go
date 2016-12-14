package reaction

import (
	"testing"

	serr "github.com/tapglue/snaas/error"
)

func TestReactionValidate(t *testing.T) {
	var (
		like  = testReactionLike()
		cases = List{
			{}, // Missign ObjectID
			{ObjectID: like.ObjectID},                                 // Missing OwnerID
			{ObjectID: like.ObjectID, OwnerID: like.OwnerID},          // Unsupported Type
			{ObjectID: like.ObjectID, OwnerID: like.OwnerID, Type: 7}, // Unsupported Type
		}
	)

	for _, r := range cases {
		if have, want := r.Validate(), serr.ErrInvalidReaction; !serr.IsInvalidReaction(have) {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
