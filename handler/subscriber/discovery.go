package subscriber

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

func (h *Handler) HandleDiscoveryRequest(ctx context.Context, discoveryReq *DiscoveryRequest) (*Response, error) {
	var stats DiscoveryStats
	for _, input := range discoveryReq.ToDescribeLogInputs() {
		paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(h.Client, input)

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to describe log groups: %w", err)
			}
			stats.RequestCount.Add(1)
			stats.LogGroupCount.Add(int64(len(page.LogGroups)))
		}
	}

	return &Response{Discovery: &stats}, nil
}
