package subscriber

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/go-logr/logr"
)

var (
	ErrMalformedEvent = errors.New("malformed cloudformation event")
)

// CloudFormationEvent adds a field which is not declared in cloudformation package
type CloudFormationEvent struct {
	*cfn.Event
}

// UnmarshalJSON provides a custom unmarshaller that allows unknown fields.
// It is critical that we respond to any CloudFormation event, otherwise stack
// install, updates and deletes will stall. In order to protect ourselves
// against unexpected fields, we succeed unmarshalling so long as we have the
// necessary elements to form a response.
func (c *CloudFormationEvent) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &c.Event); err != nil {
		return err
	}

	switch {
	case c.RequestID == "":
	case c.ResponseURL == "":
	case c.LogicalResourceID == "":
	case c.StackID == "":
	default:
		return nil
	}

	return fmt.Errorf("not a cloudformation event")
}

func makeStrSlice(item any) ([]*string, error) {
	if item == nil {
		return nil, nil
	}
	vs, ok := item.([]any)
	if !ok {
		return nil, fmt.Errorf("failed to cast %v to slice", item)
	}
	var ret []*string
	for _, v := range vs {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("failed to cast %v to string", v)
		}
		// Skip empty strings (CloudFormation passes [""] for empty parameters)
		if s != "" {
			ret = append(ret, &s)
		}
	}
	return ret, nil
}

// HandleCloudFormation triggers cleanup and discovery on CloudFormation stack updates, and cleanup on deletes
func (h *Handler) HandleCloudFormation(ctx context.Context, ev *CloudFormationEvent) (*Response, error) {
	logger := logr.FromContextOrDiscard(ctx)

	if ev == nil {
		return &Response{}, nil
	}

	logger.V(3).Info("handling cloudformation event", "requestType", ev.RequestType)

	// Prepare CloudFormation response (but don't send it yet - send after work is done)
	response := cfn.NewResponse(ev.Event)
	response.PhysicalResourceID = lambdacontext.LogStreamName
	response.Status = cfn.StatusSuccess

	var handlerResp *Response
	var handlerErr error

	switch ev.RequestType {
	case cfn.RequestUpdate:
		logger.Info("stack update detected, updating subscriptions with new patterns")

		// Extract new patterns from CloudFormation event
		var req DiscoveryRequest
		var err error
		if req.LogGroupNamePatterns, err = makeStrSlice(ev.ResourceProperties["LogGroupNamePatterns"]); err != nil {
			handlerErr = fmt.Errorf("failed to extract logGroupNamePatterns: %w", err)
			break
		}
		if req.LogGroupNamePrefixes, err = makeStrSlice(ev.ResourceProperties["LogGroupNamePrefixes"]); err != nil {
			handlerErr = fmt.Errorf("failed to extract logGroupNamePrefixes: %w", err)
			break
		}
		excludePatterns, err := makeStrSlice(ev.ResourceProperties["ExcludeLogGroupNamePatterns"])
		if err != nil {
			handlerErr = fmt.Errorf("failed to extract excludeLogGroupNamePatterns: %w", err)
			break
		}

		logger.Info("updating subscriptions with new patterns",
			"patterns", req.LogGroupNamePatterns,
			"prefixes", req.LogGroupNamePrefixes,
			"excludes", excludePatterns)

		// Build a new filter function from the CloudFormation event parameters
		// This will be used to determine which log groups should have subscriptions
		newFilter := BuildLogGroupFilter(
			ptrSliceToStrSlice(req.LogGroupNamePatterns),
			ptrSliceToStrSlice(req.LogGroupNamePrefixes),
			ptrSliceToStrSlice(excludePatterns),
		)

		// Temporarily replace the handler's filter with the CloudFormation-provided one.
		// The Handler is a singleton that persists across Lambda invocations (warm starts),
		// so we must restore the original filter after processing this CloudFormation event.
		// This ensures subsequent invocations (e.g., SQS events) continue using the
		// env-var-configured filter until the Lambda is restarted with new env vars.
		originalFilter := h.logGroupNameFilter
		h.logGroupNameFilter = newFilter
		defer func() {
			h.logGroupNameFilter = originalFilter
		}()

		// Enable FullyPrune to scan ALL log groups and remove stale subscriptions
		// that no longer match the new patterns, then subscribe matching log groups
		req.FullyPrune = true

		handlerResp, handlerErr = h.HandleDiscoveryRequest(ctx, &req)
		if handlerErr != nil {
			handlerErr = fmt.Errorf("discovery with prune failed during update: %w", handlerErr)
			break
		}

	case cfn.RequestDelete:
		// Distinguish between a true stack deletion and a resource replacement.
		// During a stack UPDATE, the Custom::Trigger resource is replaced, which triggers
		// a DELETE event for the OLD resource. We must NOT cleanup during replacement.
		//
		// Detection: During replacement, the DELETE event's PhysicalResourceId is the
		// old Lambda log stream name, while we're running in a NEW log stream.
		// If they match, it's a true stack delete. If they differ, it's a replacement.
		currentPhysicalResourceId := lambdacontext.LogStreamName
		isResourceReplacement := ev.PhysicalResourceID != "" && ev.PhysicalResourceID != currentPhysicalResourceId

		if isResourceReplacement {
			logger.Info("resource replacement detected, skipping cleanup to preserve subscriptions",
				"eventPhysicalResourceId", ev.PhysicalResourceID,
				"currentPhysicalResourceId", currentPhysicalResourceId)
			handlerResp = &Response{}
		} else {
			logger.Info("stack deletion detected, cleaning up all subscriptions",
				"eventPhysicalResourceId", ev.PhysicalResourceID,
				"currentPhysicalResourceId", currentPhysicalResourceId)

			// True stack delete - clean up all subscriptions managed by this stack
			handlerResp, handlerErr = h.HandleCleanupRequest(ctx, &CleanupRequest{
				DeleteAll: true,
			})
			if handlerErr != nil {
				handlerErr = fmt.Errorf("cleanup failed during stack delete: %w", handlerErr)
			}
		}

	default:
		handlerResp = &Response{}
	}

	// Send CloudFormation response AFTER work is complete
	if handlerErr != nil {
		response.Status = cfn.StatusFailed
		response.Reason = handlerErr.Error()
		logger.Error(handlerErr, "handler failed, sending failure response to CloudFormation")
	}

	if err := response.Send(); err != nil {
		return nil, fmt.Errorf("failed to send cloudformation response: %w", err)
	}

	// Return the handler error if there was one
	if handlerErr != nil {
		return nil, handlerErr
	}

	return handlerResp, nil
}

// ptrSliceToStrSlice converts a slice of string pointers to a slice of strings,
// skipping nil pointers and empty strings.
func ptrSliceToStrSlice(ptrs []*string) []string {
	var result []string
	for _, p := range ptrs {
		if p != nil && *p != "" {
			result = append(result, *p)
		}
	}
	return result
}
