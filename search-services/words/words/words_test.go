package words

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetMessageSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple words",
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with punctuation and uppercase",
			input:    "Hello, World! Welcome to 2024.",
			expected: []string{"hello", "world", "welcome", "to", "2024"},
		},
		{
			name:     "only symbols",
			input:    "!!! ??? ...",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMessageSlice(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"apple", "banana", "cherry"},
			expected: []string{"apple", "banana", "cherry"},
		},
		{
			name:     "with duplicates",
			input:    []string{"banana", "apple", "banana", "cherry", "apple"},
			expected: []string{"apple", "banana", "cherry"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeDuplicates(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterAndStemMessage(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		language    string
		expected    []string
		expectError bool
	}{
		{
			name:        "stemming and stop words",
			input:       []string{"the", "running", "dogs", "are", "fast"},
			language:    "english",
			expected:    []string{"run", "dog", "fast"},
			expectError: false,
		},
		{
			name:        "all stop words",
			input:       []string{"the", "is", "at", "which", "on"},
			language:    "english",
			expected:    []string{},
			expectError: false,
		},
		{
			name:        "unsupported language",
			input:       []string{"hello"},
			language:    "unknown_language",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filterAndStemMessage(tt.input, tt.language)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestNorm(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		language    string
		expected    []string
		expectError bool
	}{
		{
			name:        "full flow success",
			input:       "The running dogs are running fast!",
			language:    "english",
			expected:    []string{"dog", "fast", "run"},
			expectError: false,
		},
		{
			name:        "words and punctuation",
			input:       "I love,.. linux, Hello World!??",
			language:    "english",
			expected:    []string{"hello", "linux", "love", "world"},
			expectError: false,
		},
		{
			name:        "error in flow",
			input:       "Some words",
			language:    "invalid_lang",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Norm(tt.input, tt.language)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestNormLeaveDuplicates(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		language    string
		expected    []string
		expectError bool
	}{
		{
			name:        "full flow with duplicates",
			input:       "The running dogs are running fast!",
			language:    "english",
			expected:    []string{"run", "dog", "run", "fast"},
			expectError: false,
		},
		{
			name:        "error in flow",
			input:       "Some words",
			language:    "invalid_lang",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormLeaveDuplicates(tt.input, tt.language)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}
