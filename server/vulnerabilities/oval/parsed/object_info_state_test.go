package oval_parsed

import (
	"testing"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/require"
)

func TestObjectInfoState(t *testing.T) {
	t.Run("#EvalSoftware", func(t *testing.T) {
		t.Run("name", func(t *testing.T) {
			name := NewObjectStateString(Equals.String(), "bbq chicken")
			sut := ObjectInfoState{Operator: And, Name: &name}
			testCases := []struct {
				software mdmlab.Software
				expected bool
			}{
				{
					software: mdmlab.Software{Name: "pulled pork"},
					expected: false,
				},
				{
					software: mdmlab.Software{Name: "bbq chicken"},
					expected: true,
				},
			}

			for _, tCase := range testCases {
				r, err := sut.EvalSoftware(tCase.software)
				require.NoError(t, err)
				require.Equal(t, tCase.expected, r)
			}
		})

		t.Run("arch", func(t *testing.T) {
			arch := NewObjectStateString(Equals.String(), "x86_64")
			sut := ObjectInfoState{Operator: And, Name: &arch}
			testCases := []struct {
				software mdmlab.Software
				expected bool
			}{
				{
					software: mdmlab.Software{Arch: "i386"},
					expected: false,
				},
				{
					software: mdmlab.Software{Name: "x86_64"},
					expected: true,
				},
			}

			for _, tCase := range testCases {
				r, err := sut.EvalSoftware(tCase.software)
				require.NoError(t, err)
				require.Equal(t, tCase.expected, r)
			}
		})

		// TODO: see https://github.com/mdmlabdm/mdmlab/issues/6236 -
		// For RHEL based systems the epoch is not included in the version field

		// t.Run("epoch", func(t *testing.T) {
		// 	epoch := NewObjectStateSimpleValue(Int.String(), Equals.String(), "0")
		// 	sut := ObjectInfoState{Operator: And, Epoch: &epoch}
		// 	testCases := []struct {
		// 		software mdmlab.Software
		// 		expected bool
		// 	}{
		// 		{
		// 			software: mdmlab.Software{Version: "0:123-3"},
		// 			expected: true,
		// 		},
		// 		{
		// 			software: mdmlab.Software{Version: "123"},
		// 			expected: true,
		// 		},
		// 		{
		// 			software: mdmlab.Software{Version: ""},
		// 			expected: true,
		// 		},
		// 		{
		// 			software: mdmlab.Software{Version: "1:123"},
		// 			expected: false,
		// 		},
		// 	}

		// 	for _, tCase := range testCases {
		// 		r, err := sut.EvalSoftware(tCase.software)
		// 		require.NoError(t, err)
		// 		require.Equal(t, tCase.expected, r)
		// 	}
		// })

		t.Run("release", func(t *testing.T) {
			release := NewObjectStateSimpleValue(String.String(), Equals.String(), "0")
			sut := ObjectInfoState{Operator: And, Release: &release}
			testCases := []struct {
				software mdmlab.Software
				expected bool
			}{
				{
					software: mdmlab.Software{Version: "0:123-3"},
					expected: false,
				},
				{
					software: mdmlab.Software{Version: "123"},
					expected: false,
				},
				{
					software: mdmlab.Software{Version: "123-0"},
					expected: true,
				},
			}

			for i, tCase := range testCases {
				r, err := sut.EvalSoftware(tCase.software)
				require.NoError(t, err)
				require.Equal(t, tCase.expected, r, i)
			}
		})

		t.Run("version", func(t *testing.T) {
			version := NewObjectStateSimpleValue(String.String(), Equals.String(), "1.2")
			sut := ObjectInfoState{Operator: And, Version: &version}
			testCases := []struct {
				software mdmlab.Software
				expected bool
			}{
				{
					software: mdmlab.Software{Version: "0:123-3"},
					expected: false,
				},
				{
					software: mdmlab.Software{Version: "1.2"},
					expected: true,
				},
			}

			for i, tCase := range testCases {
				r, err := sut.EvalSoftware(tCase.software)
				require.NoError(t, err)
				require.Equal(t, tCase.expected, r, i)
			}
		})

		t.Run("evr", func(t *testing.T) {
			evr := NewObjectStateEvrString(Equals.String(), "1.2")
			sut := ObjectInfoState{Operator: And, Evr: &evr}
			testCases := []struct {
				software mdmlab.Software
				expected bool
			}{
				{
					software: mdmlab.Software{Version: "0:123-3"},
					expected: false,
				},
				{
					software: mdmlab.Software{Version: "1.2"},
					expected: true,
				},
			}

			for i, tCase := range testCases {
				r, err := sut.EvalSoftware(tCase.software)
				require.NoError(t, err)
				require.Equal(t, tCase.expected, r, i)
			}
		})

		t.Run("signature key id", func(t *testing.T) {
			sKey := NewObjectStateString(Equals.String(), "1.2")
			software := mdmlab.Software{Version: "0:123-3"}

			sut := ObjectInfoState{Operator: And, SignatureKeyId: &sKey}
			r, err := sut.EvalSoftware(software)
			require.NoError(t, err)
			require.True(t, r)
		})
	})

	t.Run("#EvalOSVersion", func(t *testing.T) {
		t.Run("name", func(t *testing.T) {
			name := NewObjectStateString(PatternMatch.String(), "^redhat-release")
			sut := ObjectInfoState{Operator: And, Name: &name}
			testCases := []struct {
				version  mdmlab.OSVersion
				expected bool
			}{
				{
					version:  mdmlab.OSVersion{Platform: "rhel", Name: "CentOS Linux 7.9.2009"},
					expected: true,
				},
				{
					version:  mdmlab.OSVersion{Platform: "amzn", Name: "Amazon Linux 2.0.0"},
					expected: true,
				},
				{
					version:  mdmlab.OSVersion{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.0.0"},
					expected: true,
				},
				{
					version:  mdmlab.OSVersion{Platform: "ubuntu", Name: "Ubuntu 22.4.0"},
					expected: false,
				},
				{
					version:  mdmlab.OSVersion{Platform: "ubuntu", Name: "Ubuntu 21.10.0"},
					expected: false,
				},
			}

			for i, tCase := range testCases {
				r, err := sut.EvalOSVersion(tCase.version)
				require.NoError(t, err)
				require.Equal(t, tCase.expected, r, i)
			}
		})

		t.Run("version", func(t *testing.T) {
			version := NewObjectStateSimpleValue(String.String(), PatternMatch.String(), `^9[^\d]`)
			sut := ObjectInfoState{Operator: And, Version: &version}
			testCases := []struct {
				version  mdmlab.OSVersion
				expected bool
			}{
				{
					version:  mdmlab.OSVersion{Platform: "rhel", Name: "CentOS Linux 7.9.2009"},
					expected: false,
				},
				{
					version:  mdmlab.OSVersion{Platform: "rhel", Name: "CentOS Linux 9.0.2020"},
					expected: true,
				},
				{
					version:  mdmlab.OSVersion{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.0.0"},
					expected: true,
				},
				{
					version:  mdmlab.OSVersion{Platform: "ubuntu", Name: "Ubuntu 22.4.0"},
					expected: false,
				},
				{
					version:  mdmlab.OSVersion{Platform: "ubuntu", Name: "Ubuntu 21.10.0"},
					expected: false,
				},
			}

			for _, tCase := range testCases {
				r, err := sut.EvalOSVersion(tCase.version)
				require.NoError(t, err)
				require.Equal(t, tCase.expected, r)
			}
		})
	})
}
