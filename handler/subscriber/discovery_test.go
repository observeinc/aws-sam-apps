package subscriber_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/google/go-cmp/cmp"

	"github.com/observeinc/aws-sam-testing/handler/handlertest"
	"github.com/observeinc/aws-sam-testing/handler/subscriber"
)

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
		DiscoveryRequest   *subscriber.DiscoveryRequest
		ExpectJSONResponse string
	}{
		{
			DiscoveryRequest: &subscriber.DiscoveryRequest{},
			/* matches:
			- /aws/hello
			- /aws/ello
			- /aws/hola
			*/
			ExpectJSONResponse: `{
				"discovery": {
					"logGroupCount": 3,
					"requestCount": 1
				}
			}`,
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
			ExpectJSONResponse: `{
				"discovery": {
					"logGroupCount": 2,
					"requestCount": 2
				}
			}`,
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
			ExpectJSONResponse: `{
				"discovery": {
					"logGroupCount": 2,
					"requestCount": 3
				}
			}`,
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
			ExpectJSONResponse: `{
				"discovery": {
					"logGroupCount": 3,
					"requestCount": 2
				}
			}`,
		},
	}

	for i, tt := range testcases {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			s, err := subscriber.New(&subscriber.Config{
				CloudWatchLogsClient: client,
			})
			if err != nil {
				t.Fatal(err)
			}

			resp, err := s.HandleDiscoveryRequest(context.Background(), tt.DiscoveryRequest)
			if err != nil {
				t.Fatal(err)
			}

			var expect bytes.Buffer
			if err := json.Compact(&expect, []byte(tt.ExpectJSONResponse)); err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(resp)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(expect.Bytes(), got); diff != "" {
				t.Error(diff)
			}
		})
	}
}
