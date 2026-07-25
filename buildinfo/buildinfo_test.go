// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsProductionBuild verifies IsProductionBuild reports true only when the
// build type equals "production" (case-insensitively, since BuildType
// lower-cases its value) and false for every other value. The match is exact:
// prefixes, superstrings, and untrimmed whitespace do not count as production.
func TestIsProductionBuild(t *testing.T) {
	cases := []struct {
		name      string
		buildType string
		want      bool
	}{
		{"exact production", "production", true},
		{"mixed case", "Production", true},
		{"upper case", "PRODUCTION", true},
		{"scrambled case", "pRoDuCtIoN", true},
		{"debug", "debug", false},
		{"dev", "dev", false},
		{"instrumentation", "instrumentation", false},
		{"empty", "", false},
		{"prefix only", "prod", false},
		{"superstring", "production-debug", false},
		{"untrimmed trailing space", "production ", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restore := SetBuildTypeForTest(tc.buildType)
			defer restore()
			assert.Equal(t, tc.want, IsProductionBuild())
		})
	}
}

// TestSetBuildTypeForTest_Restores verifies the helper restores the previous
// build type when the returned func is called, so tests cannot leak the
// global build-type override into one another.
func TestSetBuildTypeForTest_Restores(t *testing.T) {
	before := BuildType()

	restore := SetBuildTypeForTest("production")
	require.True(t, IsProductionBuild(), "build type should report production while overridden")

	restore()
	assert.Equal(t, before, BuildType(), "restore must return the build type to its prior value")
	assert.False(t, IsProductionBuild(), "restore must clear the production override")
}
