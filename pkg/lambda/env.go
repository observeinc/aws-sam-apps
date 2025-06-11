package lambda

import (
	"context"
	"fmt"
	"os"

	"github.com/sethvargo/go-envconfig"
)

// exportEnvVar ensures we export resolved values back into the environment
// This is useful for cases where we set a default in our env struct, and need
// the value to be propagated to other processes.
func exportEnvVar(_ context.Context, _, resolvedKey, _, resolvedValue string) (newValue string, stop bool, err error) {
	if err := os.Setenv(resolvedKey, resolvedValue); err != nil {
		return "", false, fmt.Errorf("failed to set environment variable %s: %w", resolvedKey, err)
	}
	return resolvedValue, false, nil
}

// ProcessEnv populates struct from environment variables.
// In doing so, any defaults for the struct are re-exported as environment
// variables.
func ProcessEnv(ctx context.Context, v any) error {
	err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:   v,
		Mutators: []envconfig.Mutator{envconfig.MutatorFunc(exportEnvVar)},
	})
	if err != nil {
		return fmt.Errorf("failed to load environment variables: %w", err)
	}
	return nil
}
