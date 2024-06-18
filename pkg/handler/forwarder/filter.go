package forwarder

import (
	"fmt"
	"regexp"
)

var globOperators = regexp.MustCompile(`(\*|\?)`)

// ObjectFilter verifies if object is intended for processing
type ObjectFilter struct {
	filters []*regexp.Regexp
}

// Allow verifies if object source should be accessed
func (o *ObjectFilter) Allow(source string) bool {
	for _, re := range o.filters {
		if re.MatchString(source) {
			return true
		}
	}
	return false
}

// NewObjectFilter initializes an ObjectFilter.
// This function will error if any bucket or object pattern are not valid glob expressions.
func NewObjectFilter(names, keys []string) (*ObjectFilter, error) {
	var obj ObjectFilter
	// TODO: for simplicity we compute the cross product of regular expressions. It
	// would be more efficient to verify buckets and object key separately, but
	// we don't expect either list to be very long.

	for _, name := range names {
		for _, key := range keys {
			source := name + "/" + key
			re, err := regexp.Compile(globOperators.ReplaceAllString(source, ".$1"))
			if err != nil {
				return nil, fmt.Errorf("failed to compile %s: %w", source, err)
			}
			obj.filters = append(obj.filters, re)
		}
	}
	return &obj, nil
}
