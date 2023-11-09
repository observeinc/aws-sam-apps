package subscriber_test

import (
	"context"
	"sync"
	"testing"

	"github.com/observeinc/aws-sam-testing/handler/handlertest"
	"github.com/observeinc/aws-sam-testing/handler/subscriber"
)

type MockQueue struct {
	values []any
	sync.Mutex
}

func (m *MockQueue) Put(_ context.Context, vs ...any) error {
	m.Lock()
	defer m.Unlock()
	m.values = append(m.values, vs...)
	return nil
}

func TestHandler(t *testing.T) {
	_, err := subscriber.New(&subscriber.Config{
		CloudWatchLogsClient: &handlertest.CloudWatchLogsClient{},
		Queue:                &MockQueue{},
	})
	if err != nil {
		t.Fatal(err)
	}
}
