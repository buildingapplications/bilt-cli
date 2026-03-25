package xcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSigningIdentities(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []SigningIdentity
	}{
		{
			name: "single development identity",
			input: `  1) ABC123DEF456789012345678901234567890ABC "Apple Development: karl@example.com (ABCDE12345)"
     1 valid identities found`,
			expected: []SigningIdentity{
				{
					Hash:   "ABC123DEF456789012345678901234567890ABC",
					Name:   "Apple Development: karl@example.com (ABCDE12345)",
					Email:  "karl@example.com",
					TeamID: "ABCDE12345",
					Type:   "Apple Development",
				},
			},
		},
		{
			name: "multiple identities mixed types",
			input: `  1) ABC123DEF456789012345678901234567890ABC "Apple Development: karl@example.com (ABCDE12345)"
  2) DEF456789012345678901234567890123456DEF "Apple Distribution: karl@example.com (ABCDE12345)"
     2 valid identities found`,
			expected: []SigningIdentity{
				{
					Hash:   "ABC123DEF456789012345678901234567890ABC",
					Name:   "Apple Development: karl@example.com (ABCDE12345)",
					Email:  "karl@example.com",
					TeamID: "ABCDE12345",
					Type:   "Apple Development",
				},
				{
					Hash:   "DEF456789012345678901234567890123456DEF",
					Name:   "Apple Distribution: karl@example.com (ABCDE12345)",
					Email:  "karl@example.com",
					TeamID: "ABCDE12345",
					Type:   "Apple Distribution",
				},
			},
		},
		{
			name:     "no identities",
			input:    "     0 valid identities found",
			expected: nil,
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name: "identity with team name in parens",
			input: `  1) AABBCCDD1234567890AABBCCDD1234567890AA "Apple Development: John Doe (XYZ789TEAM)"
     1 valid identities found`,
			expected: []SigningIdentity{
				{
					Hash:   "AABBCCDD1234567890AABBCCDD1234567890AA",
					Name:   "Apple Development: John Doe (XYZ789TEAM)",
					Email:  "John Doe",
					TeamID: "XYZ789TEAM",
					Type:   "Apple Development",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSigningIdentities(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
