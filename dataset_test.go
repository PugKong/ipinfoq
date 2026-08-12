package main

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergePrefix(t *testing.T) {
	tests := map[string]struct {
		A, B, Prefix netip.Prefix
		Merged       bool
	}{
		"merge 27": {
			A:      netip.MustParsePrefix("10.0.13.192/27"), // 10.0.13.192-10.0.13.223
			B:      netip.MustParsePrefix("10.0.13.224/27"), // 10.0.13.224-10.0.13.255
			Prefix: netip.MustParsePrefix("10.0.13.192/26"),
			Merged: true,
		},
		"not merge 27": {
			A:      netip.MustParsePrefix("10.0.13.160/27"), // 10.0.13.160-10.0.13.191
			B:      netip.MustParsePrefix("10.0.13.224/27"), // 10.0.13.224-10.0.13.255
			Prefix: netip.Prefix{},
			Merged: false,
		},
		"merge 26": {
			A:      netip.MustParsePrefix("10.0.13.128/26"), // 10.0.13.128-10.0.13.191
			B:      netip.MustParsePrefix("10.0.13.192/26"), // 10.0.13.192-10.0.13.255
			Prefix: netip.MustParsePrefix("10.0.13.128/25"),
			Merged: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			prefix, merged := mergePrefixes(test.A, test.B)
			require.Equal(t, test.Prefix, prefix)
			require.Equal(t, test.Merged, merged)
		})
	}
}
