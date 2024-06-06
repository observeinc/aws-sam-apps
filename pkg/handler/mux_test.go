package handler_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/observeinc/aws-sam-apps/pkg/handler"
)

func TestHandler(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		HandlerFuncs []any
		Checks       map[string]string
	}{
		{
			HandlerFuncs: []any{
				func(_ context.Context, _ string) (string, error) { return "string", nil },
				func(_ context.Context, _ int) (string, error) { return "int", nil },
				func(_ context.Context, _ struct{ V string }) (string, error) { return "v", nil },
				func(_ context.Context, _ struct{ W string }) (string, error) { return "w", nil },
			},
			Checks: map[string]string{
				`1`:             `"int"`,
				`"1"`:           `"string"`,
				`{"v": "test"}`: `"v"`,
				`{"w": "test"}`: `"w"`,
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			var h handler.Mux
			if err := h.Register(tc.HandlerFuncs...); err != nil {
				t.Fatal(err)
			}

			for input, output := range tc.Checks {
				result, err := h.Invoke(context.Background(), []byte(input))
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

func TestHandlerErrors(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		HandlerFuncs []any
		ExpectErr    error
	}{
		{
			// no handlers, no problem
		},
		{
			HandlerFuncs: []any{
				"1",
			},
			ExpectErr: handler.ErrHandlerType,
		},
		{
			HandlerFuncs: []any{
				func() {},
			},
			ExpectErr: handler.ErrHandlerArgsCount,
		},
		{
			HandlerFuncs: []any{
				func(int, int) {},
			},
			ExpectErr: handler.ErrHandlerRequireContext,
		},
		{
			HandlerFuncs: []any{
				func(context.Context, int) {},
			},
			ExpectErr: handler.ErrHandlerReturnCount,
		},
		{
			HandlerFuncs: []any{
				func(context.Context, int) (int, int) { return 1, 1 },
			},
			ExpectErr: handler.ErrHandlerRequireError,
		},
		{
			HandlerFuncs: []any{
				func(context.Context, int) (int, error) { return 1, nil },
				func(context.Context, int) (int, error) { return 1, nil },
			},
			ExpectErr: handler.ErrHandlerAlreadyRegistered,
		},
		{
			HandlerFuncs: []any{
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
			var h handler.Mux
			err := h.Register(tc.HandlerFuncs...)
			if diff := cmp.Diff(err, tc.ExpectErr, cmpopts.EquateErrors()); diff != "" {
				t.Error("unexpected error", diff)
			}
		})
	}
}
