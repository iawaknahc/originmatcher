package originmatcher

type wildcardMatcher struct{}

func parseWildcard(s string) (*wildcardMatcher, error) {
	if s == "*" {
		return &wildcardMatcher{}, nil
	}

	return nil, errTryNextParser
}

func (*wildcardMatcher) MatchOrigin(s string) bool {
	return true
}

func (*wildcardMatcher) String() string {
	return "*"
}
