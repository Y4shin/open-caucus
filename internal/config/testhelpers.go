package config

import (
	"os"
)

// EnvSnapshot represents saved environment variable state
type EnvSnapshot struct {
	vars map[string]*string
}

// Restore returns the environment to its previous state
func (s *EnvSnapshot) Restore() {
	for key, value := range s.vars {
		if value == nil {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, *value)
		}
	}
}

// SetTestEnv sets environment variables and returns a snapshot for restoration
// Pass map[string](*string) where nil means unset the variable
// Usage:
//
//	snapshot := SetTestEnv(map[string]*string{
//	    "PORT": stringPtr("9000"),
//	    "DEBUG": nil, // unset DEBUG
//	})
//	defer snapshot.Restore()
func SetTestEnv(vars map[string]*string) *EnvSnapshot {
	snapshot := &EnvSnapshot{
		vars: make(map[string]*string),
	}

	for key, newValue := range vars {
		// Save current value
		if currentValue, exists := os.LookupEnv(key); exists {
			snapshot.vars[key] = &currentValue
		} else {
			snapshot.vars[key] = nil
		}

		// Set new value
		if newValue == nil {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, *newValue)
		}
	}

	return snapshot
}

// stringPtr is a helper to create string pointers for test values
func stringPtr(s string) *string {
	return &s
}
