package router_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-testing/handlers/router"
)

func TestRouter(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Handlers []any
		Checks   map[string]string
	}{
		{
			Handlers: []any{
				func(_ context.Context, _ string) (string, error) { return "string", nil },
				func(_ context.Context, _ int) (string, error) { return "int", nil },
				func(_ context.Context, _ struct{ V string }) (string, error) { return "custom", nil },
			},
			Checks: map[string]string{
				`1`:             `"int"`,
				`"1"`:           `"string"`,
				`{"v": "test"}`: `"custom"`,
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			r := router.New()

			if err := r.Register(tc.Handlers...); err != nil {
				t.Fatal(err)
			}

			for input, output := range tc.Checks {
				result, err := r.Handle(context.Background(), json.RawMessage(input))
				if err != nil {
					t.Fatalf("failed to validate %s: %s", input, err)
				}

				if string(result) != output {
					t.Fatalf("mismatched return value for %s", input)
				}
			}
		})
	}
}

func TestRouterErrors(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Handlers  []any
		ExpectErr error
	}{
		{
			// no handlers, no problem
		},
		{
			Handlers: []any{
				"1",
			},
			ExpectErr: router.ErrHandlerType,
		},
		{
			Handlers: []any{
				func() {},
			},
			ExpectErr: router.ErrHandlerArgsCount,
		},
		{
			Handlers: []any{
				func(int, int) {},
			},
			ExpectErr: router.ErrHandlerRequireContext,
		},
		{
			Handlers: []any{
				func(context.Context, int) {},
			},
			ExpectErr: router.ErrHandlerReturnCount,
		},
		{
			Handlers: []any{
				func(context.Context, int) (int, int) { return 1, 1 },
			},
			ExpectErr: router.ErrHandlerRequireError,
		},
		{
			Handlers: []any{
				func(context.Context, int) (int, error) { return 1, nil },
				func(context.Context, int) (int, error) { return 1, nil },
			},
			ExpectErr: router.ErrHandlerAlreadyRegistered,
		},
		{
			Handlers: []any{
				func(context.Context, string) (int, error) { return 1, nil },
				func(context.Context, int) (int, error) { return 1, nil },
				func(context.Context, float64) (int, error) { return 1, nil },
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			r := router.New()

			err := r.Register(tc.Handlers...)
			if diff := cmp.Diff(err, tc.ExpectErr, cmpopts.EquateErrors()); diff != "" {
				t.Error("unexpected error", diff)
			}
		})
	}
}
