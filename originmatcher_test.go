package originmatcher

import (
	"testing"
)

func TestRegexp(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"localhost", `^localhost$`},
		{"*.example.com", `^[a-zA-Z][-a-zA-Z0-9]*[a-zA-Z0-9]?\.example\.com$`},
		{"a*.example.com", `^a[-a-zA-Z0-9]*[a-zA-Z0-9]?\.example\.com$`},
		{"*a.example.com", `^([a-zA-Z][-a-zA-Z0-9]*)?a\.example\.com$`},
	}
	for _, c := range cases {
		labels := parseHost(c.input)
		if labels == nil {
			t.Errorf("err\n")
		} else {
			actual := labelsToRegexpSource(labels)
			if actual != c.expected {
				t.Errorf("not eq: %v\n", actual)
			}
		}
	}
}

func TestAll(t *testing.T) {
	cases := []struct {
		input     string
		matches   []string
		unmatches []string
	}{
		{"", nil, []string{
			"http://localhost",
			"http://localhost:3000",
			"https://localhost",
			"https://localhost:3000",
			"http://example.com",
			"http://example.com:80",
			"https://example.com",
			"https://example.com:443",
			"http://www.example.com",
			"http://www.example.com:80",
			"https://www.example.com",
			"https://www.example.com:443",
		}},
		{"*", []string{
			"http://localhost",
			"http://localhost:3000",
			"https://localhost",
			"https://localhost:3000",
			"http://example.com",
			"http://example.com:80",
			"https://example.com",
			"https://example.com:443",
			"http://www.example.com",
			"http://www.example.com:80",
			"https://www.example.com",
			"https://www.example.com:443",
		}, nil},
		{"*,*", []string{
			"http://localhost",
			"http://localhost:3000",
			"https://localhost",
			"https://localhost:3000",
			"http://example.com",
			"http://example.com:80",
			"https://example.com",
			"https://example.com:443",
			"http://www.example.com",
			"http://www.example.com:80",
			"https://www.example.com",
			"https://www.example.com:443",
		}, nil},
		{"http://localhost", []string{
			"http://localhost",
			"http://localhost:80",
		}, []string{
			"http://localhost:3000",
			"https://localhost",
			"https://localhost:3000",
			"http://example.com",
			"http://example.com:80",
			"https://example.com",
			"https://example.com:443",
			"http://www.example.com",
			"http://www.example.com:80",
			"https://www.example.com",
			"https://www.example.com:443",
		}},
		{"http://localhost:3000", []string{
			"http://localhost:3000",
		}, []string{
			"http://localhost",
			"http://localhost:80",
			"https://localhost",
			"https://localhost:3000",
			"http://example.com",
			"http://example.com:80",
			"https://example.com",
			"https://example.com:443",
			"http://www.example.com",
			"http://www.example.com:80",
			"https://www.example.com",
			"https://www.example.com:443",
		}},
		{"localhost", []string{
			"http://localhost",
			"http://localhost:80",
			"https://localhost",
			"https://localhost:443",
		}, []string{
			"http://localhost:3000",
			"https://localhost:3000",
			"http://example.com",
			"http://example.com:80",
			"https://example.com",
			"https://example.com:443",
			"http://www.example.com",
			"http://www.example.com:80",
			"https://www.example.com",
			"https://www.example.com:443",
		}},
		{"localhost:3000", []string{
			"http://localhost:3000",
			"https://localhost:3000",
		}, []string{
			"http://localhost",
			"http://localhost:80",
			"https://localhost",
			"https://localhost:443",
			"http://example.com",
			"http://example.com:80",
			"https://example.com",
			"https://example.com:443",
			"http://www.example.com",
			"http://www.example.com:80",
			"https://www.example.com",
			"https://www.example.com:443",
		}},
		{"*.example.com", []string{
			"http://a.example.com",
			"http://a.example.com:80",
			"https://a.example.com",
			"https://a.example.com:443",
		}, []string{
			"http://a.example.com:81",
			"http://example.com",
		}},

		{"example.com", []string{
			"http://example.com",
			"http://example.com:80",
			"https://example.com",
			"https://example.com:443",
		}, []string{
			"http://b.example.com",
			"http://example.com:81",
		}},

		{"http://example.com", []string{
			"http://example.com",
			"http://example.com:80",
		}, []string{
			"https://example.com",
			"https://example.com:443",
		}},

		{"https://example.com", []string{
			"https://example.com",
			"https://example.com:443",
		}, []string{
			"http://example.com",
			"http://example.com:80",
		}},

		{"example.com:3000", []string{
			"http://example.com:3000",
			"https://example.com:3000",
		}, []string{
			"http://example.com",
		}},

		{"a*.*b.a*b.example.com", []string{
			"http://a.b.ab.example.com",
			"http://aa.bb.acb.example.com",
		}, []string{}},

		{"a*.*.example.com,*.example.com,example.com", []string{
			"http://www.example.com",
			"https://www.example.com",

			"http://www.example.com:80",
			"https://www.example.com:443",

			"http://example.com",
			"https://example.com",

			"http://example.com:80",
			"https://example.com:443",

			"http://a.b.example.com",
			"https://a.b.example.com",

			"http://a.b.example.com:80",
			"https://a.b.example.com:443",
		}, []string{
			"http://www.example.com:3000",
			"http://b.a.example.com",
		}},
	}
	for _, c := range cases {
		o, err := Parse(c.input)
		if err != nil {
			t.Errorf("err: %v\n", err)
		} else {
			if c.input != o.String() {
				t.Errorf("irreversible: %v %v\n", o.String(), c.input)
			}
			for _, origin := range c.matches {
				if !o.MatchOrigin(origin) {
					t.Errorf("expected match: %v %v\n", o.String(), origin)
				}
			}
			for _, origin := range c.unmatches {
				if o.MatchOrigin(origin) {
					t.Errorf("expected unmatch: %v %v\n", o.String(), origin)
				}
			}
		}
	}
}
