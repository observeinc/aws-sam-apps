package tracing_test

import (
	"fmt"
	"testing"

	"github.com/observeinc/aws-sam-apps/tracing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestOTEL(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Input     map[string]string
		Expect    map[string]string
		ExpectErr error
	}{
		{
			Input: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "hahaha",
			},
			ExpectErr: cmpopts.AnyError,
		},
		{
			Input: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost/test",
			},
			// no changes needed
			Expect: map[string]string{},
		},
		{
			Input: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://user:password@localhost/test",
			},
			Expect: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost/test",
				"OTEL_EXPORTER_OTLP_HEADERS":  "Authorization=Bearer user:password",
			},
		},
		{
			// Extend existing headers
			Input: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://user:password@localhost/test",
				"OTEL_EXPORTER_OTLP_HEADERS":  "X-Canary=true",
			},
			Expect: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost/test",
				"OTEL_EXPORTER_OTLP_HEADERS":  "X-Canary=true,Authorization=Bearer user:password",
			},
		},
		{
			// Extract resource attribute in query parameters
			Input: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost/test?deployment.environment=dev",
			},
			Expect: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost/test",
				"OTEL_RESOURCE_ATTRIBUTES":    "deployment.environment=dev",
			},
		},
		{
			// Leave unexpected query params
			Input: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost/test?timeout=1&deployment.environment=dev",
				"OTEL_RESOURCE_ATTRIBUTES":    "something=true",
			},
			Expect: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost/test?timeout=1",
				"OTEL_RESOURCE_ATTRIBUTES":    "deployment.environment=dev,something=true",
			},
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			modified := make(map[string]string)
			getter := func(k string) string {
				return tc.Input[k]
			}
			setter := func(k, v string) error {
				modified[k] = v
				return nil
			}
			err := tracing.UpdateOTELEnvVars(getter, setter)

			if diff := cmp.Diff(err, tc.ExpectErr, cmpopts.EquateErrors()); diff != "" {
				t.Error("unexpected error", diff)
			}

			if diff := cmp.Diff(modified, tc.Expect); diff != "" && tc.ExpectErr == nil {
				t.Error("unexpected result", diff)
			}
		})
	}
}
