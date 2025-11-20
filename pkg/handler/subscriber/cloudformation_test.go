package subscriber_test

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/observeinc/aws-sam-apps/pkg/handler/subscriber"
)

func TestBuildLogGroupFilter(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Name            string
		Patterns        []*string
		Prefixes        []*string
		ExcludePatterns []*string
		LogGroupName    string
		ShouldMatch     bool
	}{
		{
			Name:         "Match with prefix",
			Prefixes:     []*string{aws.String("/aws/lambda")},
			LogGroupName: "/aws/lambda/my-function",
			ShouldMatch:  true,
		},
		{
			Name:         "No match with prefix",
			Prefixes:     []*string{aws.String("/aws/lambda")},
			LogGroupName: "/aws/ecs/my-service",
			ShouldMatch:  false,
		},
		{
			Name:            "Excluded by pattern",
			Prefixes:        []*string{aws.String("/aws/lambda")},
			ExcludePatterns: []*string{aws.String("^/aws/lambda/observeinc")},
			LogGroupName:    "/aws/lambda/observeinc-forwarder",
			ShouldMatch:     false,
		},
		{
			Name:            "Not excluded",
			Prefixes:        []*string{aws.String("/aws/lambda")},
			ExcludePatterns: []*string{aws.String("^/aws/lambda/observeinc")},
			LogGroupName:    "/aws/lambda/my-app",
			ShouldMatch:     true,
		},
		{
			Name:            "Multiple exclude patterns",
			Prefixes:        []*string{aws.String("/aws/lambda")},
			ExcludePatterns: []*string{aws.String("^/aws/lambda/observeinc"), aws.String("^/aws/lambda/test")},
			LogGroupName:    "/aws/lambda/test-function",
			ShouldMatch:     false,
		},
		{
			Name:            "Multiple exclude patterns - not excluded",
			Prefixes:        []*string{aws.String("/aws/lambda")},
			ExcludePatterns: []*string{aws.String("^/aws/lambda/observeinc"), aws.String("^/aws/lambda/test")},
			LogGroupName:    "/aws/lambda/production-api",
			ShouldMatch:     true,
		},
		{
			Name:         "Wildcard pattern",
			Patterns:     []*string{aws.String("*")},
			LogGroupName: "/any/log/group",
			ShouldMatch:  true,
		},
		{
			Name:            "Wildcard with exclusion",
			Patterns:        []*string{aws.String("*")},
			ExcludePatterns: []*string{aws.String("^/aws/lambda/test")},
			LogGroupName:    "/aws/lambda/test-function",
			ShouldMatch:     false,
		},
		{
			Name:         "Exact pattern match",
			Patterns:     []*string{aws.String("/aws/lambda/my-function")},
			LogGroupName: "/aws/lambda/my-function",
			ShouldMatch:  true,
		},
		{
			Name:         "Pattern no match",
			Patterns:     []*string{aws.String("/aws/lambda/my-function")},
			LogGroupName: "/aws/lambda/other-function",
			ShouldMatch:  false,
		},
		{
			Name:         "Nil patterns",
			Patterns:     []*string{nil},
			Prefixes:     []*string{aws.String("/aws/lambda")},
			LogGroupName: "/aws/lambda/my-function",
			ShouldMatch:  true,
		},
		{
			Name:            "Empty string in exclude patterns",
			Prefixes:        []*string{aws.String("/aws/lambda")},
			ExcludePatterns: []*string{aws.String(""), aws.String("^/aws/lambda/test")},
			LogGroupName:    "/aws/lambda/my-function",
			ShouldMatch:     true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			// Use the unexported buildLogGroupFilter function via the Config.LogGroupFilter
			// which has the same logic
			cfg := &subscriber.Config{
				FilterName:     "test-filter",
				DestinationARN: "arn:aws:logs:us-west-2:123456789012:destination:test",
			}

			// Convert []*string to []string for Config
			if tc.Patterns != nil {
				for _, p := range tc.Patterns {
					if p != nil {
						cfg.LogGroupNamePatterns = append(cfg.LogGroupNamePatterns, *p)
					}
				}
			}
			if tc.Prefixes != nil {
				for _, p := range tc.Prefixes {
					if p != nil {
						cfg.LogGroupNamePrefixes = append(cfg.LogGroupNamePrefixes, *p)
					}
				}
			}
			if tc.ExcludePatterns != nil {
				for _, p := range tc.ExcludePatterns {
					if p != nil && *p != "" {
						cfg.ExcludeLogGroupNamePatterns = append(cfg.ExcludeLogGroupNamePatterns, *p)
					}
				}
			}

			filter := cfg.LogGroupFilter()
			result := filter(tc.LogGroupName)

			if result != tc.ShouldMatch {
				t.Errorf("Expected filter(%q) = %v, got %v", tc.LogGroupName, tc.ShouldMatch, result)
			}
		})
	}
}

