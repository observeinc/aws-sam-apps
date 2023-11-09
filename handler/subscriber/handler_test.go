package subscriber_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/google/go-cmp/cmp"

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

func TestHandleDiscovery(t *testing.T) {
	t.Parallel()

	client := &handlertest.CloudWatchLogsClient{
		LogGroups: []types.LogGroup{
			{LogGroupName: aws.String("/aws/hello")},
			{LogGroupName: aws.String("/aws/ello")},
			{LogGroupName: aws.String("/aws/hola")},
		},
		SubscriptionFilters: []types.SubscriptionFilter{
			{LogGroupName: aws.String("/aws/hello")},
		},
	}

	testcases := []struct {
		DiscoveryRequest *subscriber.DiscoveryRequest
		ExpectResponse   *subscriber.Response
	}{
		{
			DiscoveryRequest: &subscriber.DiscoveryRequest{},
			/* matches:
			- /aws/hello
			- /aws/ello
			- /aws/hola
			*/
			ExpectResponse: &subscriber.Response{
				DiscoveryResponse: &subscriber.DiscoveryResponse{
					RequestCount:  1,
					LogGroupCount: 3,
				},
			},
		},
		{
			DiscoveryRequest: &subscriber.DiscoveryRequest{
				LogGroupNamePrefixes: []*string{
					aws.String("/aws/he"),
					aws.String("/aws/ho"),
				},
			},
			/* matches:
			- /aws/hello
			- /aws/hola
			*/
			ExpectResponse: &subscriber.Response{
				DiscoveryResponse: &subscriber.DiscoveryResponse{
					RequestCount:  2,
					LogGroupCount: 2,
				},
			},
		},
		{
			DiscoveryRequest: &subscriber.DiscoveryRequest{
				LogGroupNamePatterns: []*string{
					aws.String("ello"),
					aws.String("foo"),
					aws.String("bar"),
				},
			},
			/* matches:
			- /aws/hello
			- /aws/ello
			*/
			ExpectResponse: &subscriber.Response{
				DiscoveryResponse: &subscriber.DiscoveryResponse{
					RequestCount:  3,
					LogGroupCount: 2,
				},
			},
		},
		{
			DiscoveryRequest: &subscriber.DiscoveryRequest{
				LogGroupNamePatterns: []*string{
					aws.String("ello"),
				},
				LogGroupNamePrefixes: []*string{
					aws.String("/aws/he"),
				},
			},
			/* matches:
			- /aws/hello
			- /aws/ello
			- /aws/hello
			*/
			ExpectResponse: &subscriber.Response{
				DiscoveryResponse: &subscriber.DiscoveryResponse{
					RequestCount:  2,
					LogGroupCount: 3,
				},
			},
		},
	}

	for i, tt := range testcases {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			s, err := subscriber.New(&subscriber.Config{
				CloudWatchLogsClient: client,
				Queue:                &MockQueue{},
			})
			if err != nil {
				t.Fatal(err)
			}

			resp, err := s.HandleDiscoveryRequest(context.Background(), tt.DiscoveryRequest)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.ExpectResponse, resp); diff != "" {
				t.Error(diff)
			}
		})
	}
}
