package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrNoHandler                = errors.New("no handler found")
	ErrHandlerType              = errors.New("handler must be a function")
	ErrHandlerAlreadyRegistered = errors.New("already registered")
	ErrHandlerArgsCount         = errors.New("handler must take 2 arguments")
	ErrHandlerRequireContext    = errors.New("first argument must be a context")
	ErrHandlerReturnCount       = errors.New("handler must return 2 values")
	ErrHandlerRequireError      = errors.New("last return value must be an error")
)

// Mux for multiple lambda handler entrypoints.
//
// This is a common helper to bridge between the convenience of declaring
// strongly typed lambda handlers, and the flexibility of routing payloads
// via the baseline Invoke method.
type Mux struct {
	Logger logr.Logger
	Tracer trace.Tracer

	handlers map[reflect.Type]reflect.Value
	sync.Mutex
}

var _ interface {
	Invoke(context.Context, []byte) ([]byte, error)
} = &Mux{}

// Register a set of lambda handlers.
func (m *Mux) Register(fns ...any) error {
	m.Lock()
	defer m.Unlock()

	if m.handlers == nil {
		m.handlers = make(map[reflect.Type]reflect.Value)
	}

	for _, f := range fns {
		handler := reflect.ValueOf(f)
		handlerType := reflect.TypeOf(f)
		if k := handlerType.Kind(); k != reflect.Func {
			return fmt.Errorf("handler kind %s: %w", k, ErrHandlerType)
		}

		if n := handlerType.NumIn(); n != 2 {
			return ErrHandlerArgsCount
		}

		if t := handlerType.In(0); !t.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			return ErrHandlerRequireContext
		}

		if n := handlerType.NumOut(); n != 2 {
			return ErrHandlerReturnCount
		}

		if t := handlerType.Out(1); !t.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			return ErrHandlerRequireError
		}

		eventType := handlerType.In(handlerType.NumIn() - 1)
		if _, ok := m.handlers[eventType]; ok {
			return fmt.Errorf("event type %s: %w", eventType, ErrHandlerAlreadyRegistered)
		}

		m.handlers[eventType] = handler
	}
	return nil
}

func (m *Mux) Invoke(ctx context.Context, payload []byte) (response []byte, err error) {
	logger := m.Logger
	if lctx, ok := lambdacontext.FromContext(ctx); ok {
		logger = m.Logger.WithValues("requestId", lctx.AwsRequestID)
		ctx = logr.NewContext(ctx, logger)
	}

	logger.V(3).Info("handling request")
	defer func() {
		if err != nil {
			logger.Error(err, "failed to process request", "payload", string(payload))
		}
	}()

	for eventType, handler := range m.handlers {
		event := reflect.New(eventType)

		dec := json.NewDecoder(bytes.NewReader(payload))
		dec.DisallowUnknownFields()

		if err := dec.Decode(event.Interface()); err != nil {
			// assume event was destined for a different handler
			continue
		}

		response := handler.Call([]reflect.Value{reflect.ValueOf(ctx), event.Elem()})

		if errVal, ok := response[1].Interface().(error); ok && errVal != nil {
			return nil, errVal
		}

		data, err := json.Marshal(response[0].Interface())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}
		return data, nil
	}
	return nil, ErrNoHandler
}
