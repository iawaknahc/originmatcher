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

var wildcardLabelRe *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z0-9*][-a-zA-Z0-9*]*[a-zA-Z0-9*]?$`)

func isDefaultPort(u *url.URL) bool {
	port := u.Port()
	if u.Scheme == "http" {
		return port == "" || port == "80"
	}
	if u.Scheme == "https" {
		return port == "" || port == "443"
	}

	return port == ""
}

func parseHost(s string) []string {
	labels := strings.Split(s, ".")

	if len(labels) == 0 {
		return nil
	}

	if len(labels) == 1 {
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

// assume s is a valid label
func labelToRegexpSource(s string) string {
	leading := "[a-zA-Z0-9]"
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

type hierarchicalMatcher struct {
	Protocol     string
	IPv4         string
	IPv6         string
	Labels       []string
	LabelsRegexp *regexp.Regexp
	Port         string
}

func parseHierarchical(s string) (*hierarchicalMatcher, error) {
	o := hierarchicalMatcher{}

	u, err := url.Parse(s)
	needParseAgain := (
	// Detect if we are parsing something like "[::1]"
	err != nil ||
		// Detect if we are parsing something like "localhost"
		(u.Scheme == "" && u.Host == "" && !strings.HasPrefix(u.Path, "/")) ||
		// Detect if we are parsing somethign like "localhost:3000"
		u.Opaque != "")
	if needParseAgain {
		u, err = url.Parse(fmt.Sprintf("https://%s", s))
	}
	if err != nil {
		return nil, errTryNextParser
	}
	if !needParseAgain {
		// We only set o.Protocol when we have no guessing.
		o.Protocol = u.Scheme
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

func (o *hierarchicalMatcher) MatchOrigin(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	if !o.matchScheme(u) {
		return false
	}

	if !o.matchHostname(u) {
		return false
	}

	if !o.matchPort(u) {
		return false
	}

	return true
}

func (o *hierarchicalMatcher) matchScheme(u *url.URL) bool {
	// If protocol is implicit, http: or https: is allowed.
	if o.Protocol == "" {
		if u.Scheme == "http" || u.Scheme == "https" {
			return true
		}
	}

	// If protocol is explicit, perform an exact match.
	if o.Protocol != "" {
		if o.Protocol == u.Scheme {
			return true
		}
	}

	return false
}

func (o *hierarchicalMatcher) matchHostname(u *url.URL) bool {
	firstRune, _ := utf8.DecodeRuneInString(u.Hostname())
	if firstRune == utf8.RuneError {
		return false
	}

	hostname := u.Hostname()
	if ip := net.ParseIP(hostname); ip != nil {
		if ip.To4() != nil {
			if hostname != o.IPv4 {
				return false
			}
		} else {
			if hostname != o.IPv6 {
				return false
			}
		}
	} else {
		if o.LabelsRegexp == nil || !o.LabelsRegexp.MatchString(u.Hostname()) {
			return false
		}
	}

	return true
}

func (o *hierarchicalMatcher) matchPort(u *url.URL) bool {
	if o.isDefaultPort() {
		if isDefaultPort(u) {
			return true
		}
	}

	if o.Port == u.Port() {
		return true
	}

	return false
}

func (o *hierarchicalMatcher) isDefaultPort() bool {
	if o.Port == "" {
		return true
	}
	if o.Protocol == "http" {
		return o.Port == "" || o.Port == "80"
	}
	if o.Protocol == "https" {
		return o.Port == "" || o.Port == "443"
	}

	return false
}

func (o *hierarchicalMatcher) String() string {
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
