package tagger

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
)

// ResourceGroupsTaggingClient wraps the AWS SDK tagging API client.
type ResourceGroupsTaggingClient struct {
	client *resourcegroupstaggingapi.Client
}

func NewResourceGroupsTaggingClient(client *resourcegroupstaggingapi.Client) *ResourceGroupsTaggingClient {
	return &ResourceGroupsTaggingClient{client: client}
}

// GetResourcesByType returns a map of resource identifiers to their tags for
// the given resource type. The resource identifier is extracted from the ARN
// (typically the last segment).
func (c *ResourceGroupsTaggingClient) GetResourcesByType(ctx context.Context, resourceType string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)

	input := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: []string{resourceType},
	}

	paginator := resourcegroupstaggingapi.NewGetResourcesPaginator(c.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("paginate GetResources: %w", err)
		}
		for _, resource := range page.ResourceTagMappingList {
			id := ExtractResourceID(aws.ToString(resource.ResourceARN))
			tags := tagsToMap(resource.Tags)
			if len(tags) > 0 {
				result[id] = tags
			}
		}
	}

	return result, nil
}

func tagsToMap(tags []types.Tag) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		m[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}
	return m
}
