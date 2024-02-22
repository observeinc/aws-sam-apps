package override

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/mitchellh/mapstructure"
)

var (
	reIdentifier = regexp.MustCompile("^([a-zA-Z][a-zA-Z0-9/]+)?$")
	errIDFormat  = errors.New("malformed id")
)

// Filter input object.
type Filter struct {
	Source          *regexp.Regexp `mapstructure:"source"`
	ContentType     *regexp.Regexp `mapstructure:"content-type"`
	ContentEncoding *regexp.Regexp `mapstructure:"content-encoding"`
}

// Match input object.
func (f *Filter) Match(input *s3.CopyObjectInput) bool {
	if f.Source != nil && !f.Source.MatchString(aws.ToString(input.CopySource)) {
		return false
	}
	if f.ContentType != nil && !f.ContentType.MatchString(aws.ToString(input.ContentType)) {
		return false
	}
	if f.ContentEncoding != nil && !f.ContentEncoding.MatchString(aws.ToString(input.ContentEncoding)) {
		return false
	}
	return true
}

// Action to be applied when copying an object.
type Action struct {
	// Content Type override
	ContentType     *string `mapstructure:"content-type"`
	ContentEncoding *string `mapstructure:"content-encoding"`
}

func (a *Action) Apply(_ context.Context, input *s3.CopyObjectInput) bool {
	if a.ContentType != nil {
		input.ContentType = a.ContentType
		input.MetadataDirective = types.MetadataDirectiveReplace
	}
	if a.ContentEncoding != nil {
		input.ContentEncoding = a.ContentEncoding
		input.MetadataDirective = types.MetadataDirectiveReplace
	}
	return true
}

// Rule contains a filter and override.
type Rule struct {
	ID       string `mapstructure:"id"`       // ID is a machine readable identifier
	Match    Filter `mapstructure:"match"`    // Filter on input object
	Override Action `mapstructure:"override"` // Action to apply if matched
	Continue bool   `mapstructure:"continue"` // Whether rule is terminal
}

func (r *Rule) Apply(ctx context.Context, input *s3.CopyObjectInput) bool {
	if r.Match.Match(input) {
		return r.Override.Apply(ctx, input)
	}
	return false
}

// Validate rule is sane.
func (r *Rule) Validate() error {
	if !reIdentifier.MatchString(r.ID) {
		return fmt.Errorf("%w: %q does not match allowed format %q", errIDFormat, r.ID, reIdentifier.String())
	}
	return nil
}

// UnmarshalText populates a rule from rudimentary key value representation.
func (r *Rule) UnmarshalText(text []byte) error {
	s := string(text)
	if s == "" {
		// treat empty values as noops
		// this overcomes a cloudformation limitation in concatenating
		// multiple values, which can result in trailing commas
		r.Match.Source = regexp.MustCompile("^$")
		return nil
	}

	split := strings.SplitN(s, defaultDelimiter, 2)
	if len(split) != 2 {
		return fmt.Errorf("error parsing %q: %w", s, errMissingDelimiter)
	}

	pattern, contentType := split[0], split[1]
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("failed to parse regular expression %q: %w", pattern, err)
	}

	r.Match.Source = re
	r.Override.ContentType = &contentType
	return nil
}

// UnmarshalYAML handles compiling regexp.
func (r *Rule) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v any
	if err := unmarshal(&v); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		DecodeHook:  stringToRegexpFunc,
		Result:      &r,
	})
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("failed to decode rule: %w", err)
	}

	return nil
}

func stringToRegexpFunc(f reflect.Type, t reflect.Type, data any) (any, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t != reflect.TypeOf(regexp.Regexp{}) {
		return data, nil
	}
	s, _ := data.(string)
	re, err := regexp.Compile(s)
	if err != nil {
		return data, fmt.Errorf("hook error: %w", err)
	}
	return re, nil
}
