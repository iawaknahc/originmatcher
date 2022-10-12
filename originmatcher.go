package originmatcher

import (
	"fmt"
	"strings"
)

type matcher interface {
	MatchOrigin(s string) bool
	String() string
}

type T struct {
	matchers []matcher
}

// String returns the original string that was parsed into t.
func (t *T) String() string {
	specs := []string{}
	for _, matcher := range t.matchers {
		specs = append(specs, matcher.String())
	}
	return strings.Join(specs, ",")
}

// MatchOrigin tells whether s is an allowed origin.
// s typically should be the value of HTTP header "Origin".
// MatchOrigin is lenient that extra userinfo, path , query or fragment in s are ignored silently.
func (t *T) MatchOrigin(s string) bool {
	for _, o := range t.matchers {
		if o.MatchOrigin(s) {
			return true
		}
	}
	return false
}

func parseSingle(s string) (matcher, error) {
	if s == "*" {
		return wildcardMatcher{}, nil
	}

	return parseHierarchical(s)
}

// Parse parses s into T, where s is comma-separated origin specs.
func Parse(s string) (*T, error) {
	// strings.Split("", ",") == [""]
	// which results in invalid spec
	if s == "" {
		return &T{
			matchers: []matcher{},
		}, nil
	}
	return New(strings.Split(s, ","))
}

// New creates a T from a slice of origin spec.
// An origin spec consist of a mandatory host, with optionally scheme and port.
// As a special case, "*" matches any origin.
// An origin spec is lenient that extra userinfo, path, query or fragment are ignored silently.
func New(specs []string) (*T, error) {
	t := &T{
		matchers: []matcher{},
	}
	for _, spec := range specs {
		o, err := parseSingle(spec)
		if err != nil {
			return nil, err
		}
		t.matchers = append(t.matchers, o)
	}
	return t, nil
}

// CheckValidSpecStrict checks if spec is valid and does not contain extra information.
func CheckValidSpecStrict(spec string) (err error) {
	o, err := parseSingle(spec)
	if err != nil {
		return
	}

	if o.String() != spec {
		err = fmt.Errorf("%v is not strict", spec)
		return
	}

	return
}
