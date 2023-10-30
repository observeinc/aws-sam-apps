package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
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

type Router struct {
	handlers map[reflect.Type]reflect.Value
	sync.Mutex
}

// Register a lambda handler.
func (r *Router) Register(fs ...any) error {
	r.Lock()
	defer r.Unlock()
	for _, f := range fs {
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
		if _, ok := r.handlers[eventType]; ok {
			return fmt.Errorf("event type %s: %w", eventType, ErrHandlerAlreadyRegistered)
		}

		r.handlers[eventType] = handler
	}
	return nil
}

func (r *Router) Handle(ctx context.Context, v json.RawMessage) (json.RawMessage, error) {
	for eventType, handler := range r.handlers {
		event := reflect.New(eventType)

		if err := json.Unmarshal(v, event.Interface()); err != nil {
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
		return json.RawMessage(data), nil
	}
	return nil, ErrNoHandler
}

func New() *Router {
	return &Router{
		handlers: make(map[reflect.Type]reflect.Value),
	}
}
