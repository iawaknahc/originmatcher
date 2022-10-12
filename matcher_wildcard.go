package originmatcher

type wildcardMatcher struct{}

func (wildcardMatcher) MatchOrigin(s string) bool {
	return true
}

func (wildcardMatcher) String() string {
	return "*"
}
