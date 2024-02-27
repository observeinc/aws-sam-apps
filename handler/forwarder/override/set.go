package override

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/go-logr/logr"
)

var (
	errDuplicate        = errors.New("duplicate ID")
	errMissingDelimiter = errors.New("missing delimiter")
	defaultDelimiter    = "="
)

// Set is a sequence of rules.
type Set struct {
	Logger logr.Logger
	Rules  []*Rule
}

func (s *Set) Apply(ctx context.Context, input *s3.CopyObjectInput) (modified bool) {
	for i, rule := range s.Rules {
		if rule.Apply(ctx, input) {
			modified = true
			id := rule.ID
			if id == "" {
				id = fmt.Sprintf("%d", i)
			}
			s.Logger.Info("applied rule", "id", id)
			if !rule.Continue {
				return
			}
		}
	}
	return
}

// Validate rules do not have duplicate ids.
func (s *Set) Validate() error {
	seen := make(map[string]struct{}, len(s.Rules))
	for i, rule := range s.Rules {
		id := fmt.Sprintf("%d", i)
		if rule.ID != "" {
			id = rule.ID
		}

		if _, dupe := seen[id]; dupe {
			return fmt.Errorf("rule %q: %w", id, errDuplicate)
		}
		seen[rule.ID] = struct{}{}
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("rule %q: %w", id, err)
		}
	}
	return nil
}

type Sets []*Set

func (ss Sets) Apply(ctx context.Context, input *s3.CopyObjectInput) (modified bool) {
	for _, s := range ss {
		if s.Apply(ctx, input) {
			return true
		}
	}
	return false
}
