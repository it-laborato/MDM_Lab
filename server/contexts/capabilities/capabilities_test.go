package capabilities

import (
	"context"
	"net/http"
	"testing"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/require"
)

func TestCapabilitiesExist(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  mdmlab.CapabilityMap
	}{
		{"empty", "", mdmlab.CapabilityMap{}},
		{"one", "test", mdmlab.CapabilityMap{mdmlab.Capability("test"): struct{}{}}},
		{
			"many",
			"test,foo,bar",
			mdmlab.CapabilityMap{
				mdmlab.Capability("test"): struct{}{},
				mdmlab.Capability("foo"):  struct{}{},
				mdmlab.Capability("bar"):  struct{}{},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := http.Request{
				Header: http.Header{mdmlab.CapabilitiesHeader: []string{tt.in}},
			}
			ctx := NewContext(context.Background(), &r)
			mp, ok := FromContext(ctx)
			require.True(t, ok)
			require.Equal(t, tt.out, mp)
		})
	}
}

func TestCapabilitiesNotExist(t *testing.T) {
	mp, ok := FromContext(context.Background())
	require.False(t, ok)
	require.Nil(t, mp)
}
