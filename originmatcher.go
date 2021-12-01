package originmatcher

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

var portRe *regexp.Regexp = regexp.MustCompile(`^:(\d{1,5})$`)
var wildcardLabelRe *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z0-9*][-a-zA-Z0-9*]*[a-zA-Z0-9*]?$`)

type origin struct {
	Protocol     string
	IPv4         string
	IPv6         string
	Labels       []string
	LabelsRegexp *regexp.Regexp
	Port         string
}

func (o *origin) isSpecialCase() bool {
	return o.Protocol == "" && o.Port == "" && len(o.Labels) == 1 && o.Labels[0] == "*"
}

func (o *origin) MatchOrigin(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if o.isSpecialCase() {
		return true
	}

	if o.Protocol != "" {
		if o.Protocol != u.Scheme {
			return false
		}
	} else {
		if u.Scheme != "http" && u.Scheme != "https" {
			return false
		}
	}

	firstRune, _ := utf8.DecodeRuneInString(u.Hostname())
	if firstRune == utf8.RuneError {
		return false
	}
	if strings.HasPrefix(u.Host, "[") {
		// IPv6
		if u.Hostname() != o.IPv6 {
			return false
		}
	} else if firstRune >= '0' && firstRune <= '9' {
		// IPv4
		if u.Hostname() != o.IPv4 {
			return false
		}
	} else {
		if o.LabelsRegexp == nil || !o.LabelsRegexp.MatchString(u.Hostname()) {
			return false
		}
	}

	actualPort := u.Port()
	expectedPort := o.Port
	if u.Scheme == "http" {
		if actualPort == "" {
			actualPort = "80"
		}
		if expectedPort == "" {
			expectedPort = "80"
		}
	} else if u.Scheme == "https" {
		if actualPort == "" {
			actualPort = "443"
		}
		if expectedPort == "" {
			expectedPort = "443"
		}
	}
	if actualPort != expectedPort {
		return false
	}

	return true
}

func (o *origin) String() string {
	out := ""
	if o.Protocol != "" {
		out += o.Protocol + "://"
	}
	if o.IPv4 != "" {
		out += o.IPv4
	} else if o.IPv6 != "" {
		out += "[" + o.IPv6 + "]"
	} else {
		out += strings.Join(o.Labels, ".")
	}
	if o.Port != "" {
		out += ":" + o.Port
	}
	return out
}

type T struct {
	origins []origin
}

// String returns the original string that was parsed into t.
func (t *T) String() string {
	specs := []string{}
	for _, origin := range t.origins {
		specs = append(specs, origin.String())
	}
	return strings.Join(specs, ",")
}

// MatchOrigin tells whether s is an allowed origin.
// s typically should be the value of HTTP header "Origin".
// MatchOrigin is lenient that extra userinfo, path , query or fragment in s are ignored silently.
func (t *T) MatchOrigin(s string) bool {
	for _, o := range t.origins {
		if o.MatchOrigin(s) {
			return true
		}
	}
	return false
}

// assume s is a valid label
func labelToRegexpSource(s string) string {
	leading := "[a-zA-Z]"
	middle := "[-a-zA-Z0-9]*"
	trailing := "[a-zA-Z0-9]?"
	b := []byte(s)
	length := len(b)
	i := bytes.IndexRune(b, '*')
	if i < 0 {
		return s
	}
	if i == 0 {
		// "*"
		if i == length-1 {
			return leading + middle + trailing
		}
		// "*a"
		return "(" + leading + middle + ")?" + string(b[i+1:])
	} else {
		// "a*"
		if i == length-1 {
			return string(b[:i]) + middle + trailing
		}
		// "a*b"
		return string(b[:i]) + middle + string(b[i+1:])
	}
}

func labelsToRegexpSource(a []string) string {
	sources := []string{}
	for _, label := range a {
		sources = append(sources, labelToRegexpSource(label))
	}
	return "^" + strings.Join(sources, `\.`) + "$"
}

func parseHost(s string) []string {
	labels := strings.Split(s, ".")

	if len(labels) == 0 {
		return nil
	}

	if len(labels) == 1 {
		// Allow special case "*"
		if labels[0] == "*" {
			return labels
		}
		// Otherwise disallow any "*" in host
		// Thus allow "localhost"
		if strings.ContainsRune(labels[0], '*') {
			return nil
		}
		return labels
	}

	length := len(labels)
	expectNoMoreStar := false
	// Iterate labels from left to right
	for i, label := range labels {
		starCount := strings.Count(label, "*")
		if starCount > 0 && expectNoMoreStar {
			return nil
		}
		if i == length-1 {
			// The last label must have no stars
			if starCount > 0 {
				return nil
			}
		} else {
			// Other labels can have at most 1 star
			if starCount > 1 {
				return nil
			}
		}
		// If this label has no star, then we expect subsequent
		// labels contain no star.
		if starCount == 0 {
			expectNoMoreStar = true
		}
		if !wildcardLabelRe.MatchString(label) {
			return nil
		}
	}

	return labels
}

func parseSingle(s string) (*origin, error) {
	o := origin{}

	if strings.HasPrefix(s, "http://") {
		o.Protocol = "http"
		s = strings.TrimPrefix(s, "http://")
	} else if strings.HasPrefix(s, "https://") {
		o.Protocol = "https"
		s = strings.TrimPrefix(s, "https://")
	}

	u, err := url.Parse(fmt.Sprintf("https://%v", s))
	if err != nil {
		return nil, err
	}

	o.Port = u.Port()

	hostname := u.Hostname()
	if ip := net.ParseIP(hostname); ip != nil {
		if ip.To4() != nil {
			o.IPv4 = hostname
		} else {
			o.IPv6 = hostname
		}
	} else {
		labels := parseHost(hostname)
		if labels == nil {
			return nil, fmt.Errorf("invalid host: %v", hostname)
		}
		o.Labels = labels
		re, err := regexp.Compile(labelsToRegexpSource(labels))
		if err != nil {
			return nil, fmt.Errorf("internal error: %v", err)
		}
		o.LabelsRegexp = re
	}

	return &o, nil
}

// Parse parses s into T, where s is comma-separated origin specs.
func Parse(s string) (*T, error) {
	// strings.Split("", ",") == [""]
	// which results in invalid spec
	if s == "" {
		return &T{
			origins: []origin{},
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
		origins: []origin{},
	}
	for _, spec := range specs {
		o, err := parseSingle(spec)
		if err != nil {
			return nil, err
		}
		t.origins = append(t.origins, *o)
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
