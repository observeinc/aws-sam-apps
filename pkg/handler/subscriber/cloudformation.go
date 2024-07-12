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
		ret = append(ret, &s)
	}
	return ret, nil
}

// HandleCloudFormation triggers discovery on CloudFormation stack updates
func (h *Handler) HandleCloudFormation(ctx context.Context, ev *CloudFormationEvent) (*Response, error) {
	logger := logr.FromContextOrDiscard(ctx)

	if ev == nil {
		return &Response{}, nil
	}

	logger.V(3).Info("handling cloudformation event", "requestType", ev.RequestType)

	response := cfn.NewResponse(ev.Event)
	response.PhysicalResourceID = lambdacontext.LogStreamName
	response.Status = cfn.StatusSuccess
	err := response.Send()
	if err != nil {
		return nil, fmt.Errorf("failed to send cloudformation response: %w", err)
	}

	if ev.RequestType == cfn.RequestUpdate {
		var req DiscoveryRequest
		if req.LogGroupNamePatterns, err = makeStrSlice(ev.ResourceProperties["LogGroupNamePatterns"]); err != nil {
			return nil, fmt.Errorf("failed to extract logGroupNamePatterns: %w", err)
		}
		if req.LogGroupNamePrefixes, err = makeStrSlice(ev.ResourceProperties["LogGroupNamePrefixes"]); err != nil {
			return nil, fmt.Errorf("failed to extract logGroupNamePrefixes: %w", err)
		}
		return h.HandleDiscoveryRequest(ctx, &req)
	}

	return &Response{}, nil
}
