package license

import (
	"context"
	"testing"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/require"
)

func TestIsPremium(t *testing.T) {
	cases := []struct {
		desc string
		ctx  context.Context
		want bool
	}{
		{"no license", context.Background(), false},
		{"free license", NewContext(context.Background(), &mdmlab.LicenseInfo{Tier: mdmlab.TierFree}), false},
		{"premium license", NewContext(context.Background(), &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium}), true},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := IsPremium(c.ctx)
			require.Equal(t, c.want, got)
		})
	}
}
