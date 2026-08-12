package workflows

import (
	"testing"
)

func TestExtractDelaySeconds(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		expected int64
	}{
		{
			name:     "duration and unit minutes",
			params:   map[string]interface{}{"duration": 10, "unit": "minutes"},
			expected: 600,
		},
		{
			name:     "duration and unit hours",
			params:   map[string]interface{}{"duration": 24, "unit": "hours"},
			expected: 86400,
		},
		{
			name:     "duration and unit days",
			params:   map[string]interface{}{"duration": 2, "unit": "days"},
			expected: 172800,
		},
		{
			name:     "duration string 24h",
			params:   map[string]interface{}{"duration": "24h"},
			expected: 86400,
		},
		{
			name:     "duration string with word unit",
			params:   map[string]interface{}{"duration": "10 minutes"},
			expected: 600,
		},
		{
			name:     "duration_seconds explicit key",
			params:   map[string]interface{}{"duration_seconds": 300},
			expected: 300,
		},
		{
			name:     "delay_ms explicit key",
			params:   map[string]interface{}{"delay_ms": 5000},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDelaySeconds(tt.params)
			if got != tt.expected {
				t.Errorf("extractDelaySeconds() = %d, expected %d", got, tt.expected)
			}
		})
	}
}
