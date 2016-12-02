package platform

import (
	"testing"

	pErr "github.com/tapglue/snaas/error"
)

func TestValidate(t *testing.T) {
	var (
		p  = testPlatform()
		ps = List{
			{},                                                 // Missing ARN
			{ARN: p.ARN},                                       // Missing Ecosystem
			{ARN: p.ARN, Ecosystem: 4},                         // Unsupported Ecosystem
			{ARN: p.ARN, Ecosystem: p.Ecosystem},               // Missing Name
			{ARN: p.ARN, Ecosystem: p.Ecosystem, Name: p.Name}, // Missing Scheme
		}
	)

	for _, p := range ps {
		if have, want := p.Validate(), pErr.ErrInvalidPlatform; !pErr.IsInvalidPlatform(have) {
			t.Errorf("have %v, want %v", have, want)
		}
	}
}
