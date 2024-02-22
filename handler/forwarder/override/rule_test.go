package override_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"

	"github.com/observeinc/aws-sam-apps/handler/forwarder/override"
)

// helper function so that we can indent config files and make things more
// readable in test code.
func trimLeadingWhitespace(in string) (out string) {
	delim := "\n"
	lines := strings.Split(in, delim)

	start := 0
	// skip leading empty lines
	for strings.TrimSpace(lines[start]) == "" {
		start++
	}

	leadingWhitespace := strings.TrimSuffix(lines[start], strings.TrimSpace(lines[start]))
	for i := start; i < len(lines); i++ {
		out += strings.TrimPrefix(lines[i], leadingWhitespace) + delim
	}
	return
}

var regexpComparer = cmp.Comparer(func(x, y *regexp.Regexp) bool {
	return x == y || (x != nil && y != nil && x.String() == y.String())
})

func TestRuleText(t *testing.T) {
	testcases := []struct {
		Text        string
		ExpectError *regexp.Regexp
		Expect      *override.Rule
	}{
		{
			Text:        "nonono",
			ExpectError: regexp.MustCompile(`error parsing "nonono": missing delimiter`),
		},
		{
			Text:        `\`,
			ExpectError: regexp.MustCompile(regexp.QuoteMeta(`error parsing "\\": missing delimiter`)),
		},
		{
			Text: ``,
			Expect: &override.Rule{
				Match: override.Filter{
					Source: regexp.MustCompile("^$"),
				},
			},
		},
		{
			Text: `.*=application/json`,
			Expect: &override.Rule{
				Match: override.Filter{
					Source: regexp.MustCompile(".*"),
				},
				Override: override.Action{
					ContentType: ptr("application/json"),
				},
			},
		},
	}

	for _, tt := range testcases {
		var rule override.Rule
		err := rule.UnmarshalText([]byte(tt.Text))
		if err == nil {
			err = rule.Validate()
		}
		switch {
		case err == nil && tt.ExpectError == nil:
			if diff := cmp.Diff(&rule, tt.Expect, regexpComparer); diff != "" {
				t.Error("rules do not match", diff)
			}
			// ok
		case err == nil && tt.ExpectError != nil:
			t.Error("expected error")
		case tt.ExpectError != nil && !tt.ExpectError.MatchString(err.Error()):
			t.Error("error does not match expected:", err)
		case err != nil:
			continue
		}
	}
}

func TestRuleYAML(t *testing.T) {
	testcases := []struct {
		YAML        string
		ExpectError *regexp.Regexp
	}{
		{
			YAML: trimLeadingWhitespace(`
			---
			id: test
			match:
			  source: '\'
			`),
			ExpectError: regexp.MustCompile("error decoding 'match.source': hook error: error parsing regexp: trailing backslash at end of expression"),
		},
		{
			YAML: trimLeadingWhitespace(`
			---
			id: '!#!'
			match:
			  source: '.*'
			`),
			ExpectError: regexp.MustCompile("malformed id: \"!#!\" does not match allowed format"),
		},
		{
			// ID should not be mandatory
			YAML: trimLeadingWhitespace(`
			---
			match:
			`),
		},
		{
			YAML: trimLeadingWhitespace(`
			---
			id: 'hello'
			match:
			  source: '.*'
			`),
		},
	}

	for _, tt := range testcases {
		var rule *override.Rule
		err := yaml.Unmarshal([]byte(tt.YAML), &rule)
		if err == nil {
			err = rule.Validate()
		}
		switch {
		case err == nil && tt.ExpectError == nil:
			// ok
		case err == nil && tt.ExpectError != nil:
			t.Error("expected error")
		case tt.ExpectError != nil && !tt.ExpectError.MatchString(err.Error()):
			t.Error("error does not match expected:", err)
		case err != nil:
			continue
		}
	}
}
