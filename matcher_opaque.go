package originmatcher

import (
	"net/url"
)

type opaqueMatcher struct {
	URL string
}

func parseOpaque(s string) (*opaqueMatcher, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, errTryNextParser
	}

	if u.Scheme != "" && u.Opaque != "" {
		return &opaqueMatcher{
			URL: s,
		}, nil
	}

	return nil, errTryNextParser
}

func (o *opaqueMatcher) MatchOrigin(s string) bool {
	return o.URL == s
}

func (o *opaqueMatcher) String() string {
	return o.URL
}
