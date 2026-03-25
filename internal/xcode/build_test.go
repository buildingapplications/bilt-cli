package xcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSchemes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "typical output",
			input: `Information about workspace "MyApp":
    Schemes:
        MyApp
        MyAppTests
`,
			expected: []string{"MyApp", "MyAppTests"},
		},
		{
			name: "single scheme",
			input: `Information about workspace "App":
    Schemes:
        App
`,
			expected: []string{"App"},
		},
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name: "no schemes section",
			input: `Information about workspace "App":
    Targets:
        App
`,
			expected: nil,
		},
		{
			name: "schemes followed by another section",
			input: `Information about workspace "MyApp":
    Schemes:
        MyApp
        MyAppTests

    Build Configurations:
        Debug
        Release
`,
			expected: []string{"MyApp", "MyAppTests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSchemes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPickAppScheme(t *testing.T) {
	tests := []struct {
		name      string
		schemes   []string
		workspace string
		expected  string
	}{
		{
			name:      "exact workspace name match",
			schemes:   []string{"EXConstants", "React-Core", "RushVault", "Yoga"},
			workspace: "RushVault.xcworkspace",
			expected:  "RushVault",
		},
		{
			name:      "filters out all library schemes",
			schemes:   []string{"EXConstants", "React-Core", "hermes-engine", "Yoga", "MyApp"},
			workspace: "MyApp.xcworkspace",
			expected:  "MyApp",
		},
		{
			name:      "only candidate after filtering",
			schemes:   []string{"React-Core", "RNScreens", "ExpoModulesCore", "FitApp"},
			workspace: "SomeOther.xcworkspace",
			expected:  "FitApp",
		},
		{
			name:      "simple workspace",
			schemes:   []string{"App", "AppTests"},
			workspace: "App.xcworkspace",
			expected:  "App",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PickAppScheme(tt.schemes, tt.workspace)
			assert.Equal(t, tt.expected, result)
		})
	}
}
