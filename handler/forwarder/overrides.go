package forwarder

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	errMissingDelimiter = errors.New("missing delimiter")
	defaultDelimiter    = "="
)

type Matcher interface {
	Match(string) string
}

type matches []Matcher

func (ms *matches) Match(s string) string {
	for _, m := range *ms {
		if v := m.Match(s); v != "" {
			return v
		}
	}
	return ""
}

type contentTypeOverride struct {
	Pattern     *regexp.Regexp
	ContentType string
}

func (c *contentTypeOverride) Match(s string) string {
	if c.Pattern.MatchString(s) {
		return c.ContentType
	}
	return ""
}

func NewContentTypeOverrides(kvs []string, delimiter string) (Matcher, error) {
	var m matches
	for _, pair := range kvs {
		split := strings.SplitN(pair, delimiter, 2)
		if len(split) != 2 {
			return nil, fmt.Errorf("error parsing %q: %w", pair, errMissingDelimiter)
		}

		pattern, contentType := split[0], split[1]
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}

		m = append(m, &contentTypeOverride{
			Pattern:     re,
			ContentType: contentType,
		})
	}
	return &m, nil
}
