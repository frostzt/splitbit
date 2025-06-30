package internals

import (
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name         string
		config       SplitbitConfig
		expectsError bool
		expects      string
	}{
		{
			name: "with a valid configuration",
			config: SplitbitConfig{
				Name:      "Splitbit Config",
				Algorithm: "round-robin",
				Scheme:    "tcp",
				Backends: []BackendConfig{
					{
						Name:        "test",
						Host:        "127.0.0.1",
						Port:        8000,
						Weight:      2,
						HealthCheck: "/health",
					},
				},
			},
			expectsError: false,
			expects:      "",
		},
		{
			name: "with a invalid configuration",
			config: SplitbitConfig{
				Name:      "Splitbit Config",
				Algorithm: "round-batman",
				Scheme:    "tcp",
				Backends: []BackendConfig{
					{
						Name:        "test",
						Host:        "127.0.0.1",
						Port:        8000,
						Weight:      2,
						HealthCheck: "/health",
					},
				},
			},
			expectsError: true,
			expects:      "are supported as algorithm",
		},
		{
			name: "without backends provided",
			config: SplitbitConfig{
				Name:      "Splitbit Config",
				Algorithm: "round-robin",
				Scheme:    "tcp",
				Backends:  []BackendConfig{},
			},
			expectsError: true,
			expects:      "at least one backend is required",
		},
		{
			name: "with an invalid backend port",
			config: SplitbitConfig{
				Name:      "Splitbit Config",
				Algorithm: "round-robin",
				Scheme:    "tcp",
				Backends: []BackendConfig{
					{
						Name:        "test",
						Host:        "127.0.0.1",
						Port:        80_000,
						Weight:      2,
						HealthCheck: "/health",
					},
				},
			},
			expectsError: true,
			expects:      "a valid port is required",
		},
		{
			name: "with an invalid weight",
			config: SplitbitConfig{
				Name:      "Splitbit Config",
				Algorithm: "round-robin",
				Scheme:    "tcp",
				Backends: []BackendConfig{
					{
						Name:        "test",
						Host:        "127.0.0.1",
						Port:        8000,
						Weight:      -1,
						HealthCheck: "/health",
					},
				},
			},
			expectsError: true,
			expects:      "weight must be a positive integer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()

			if err != nil && !test.expectsError {
				t.Error(err)
			}

			if err == nil && test.expectsError {
				t.Error("Expected an error")
			}

			if err != nil && test.expectsError {
				if test.expects != "" && !strings.Contains(err.Error(), test.expects) {
					t.Errorf("Expected error to contain %q, got %q", test.expects, err.Error())
				}
			}
		})
	}
}
