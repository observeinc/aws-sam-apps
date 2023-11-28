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
	for _, value := range kvs {
		index := strings.Index(value, delimiter)
		if index == -1 {
			return nil, fmt.Errorf("error parsing %q: %w", value, errMissingDelimiter)
		}
		re, err := regexp.Compile(value[:index])
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", value[:index], err)
		}
		m = append(m, &contentTypeOverride{
			Pattern:     re,
			ContentType: value[index+1:],
		})
	}
	return &m, nil
}
