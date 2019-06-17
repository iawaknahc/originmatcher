# Match HTTP Origin

Read the test case to see what it does.

[Documentation](https://godoc.org/github.com/iawaknahc/originmatcher)

# Example

Suppose you host your frontend on `https://www.example.com` and `https://example.com`, while your backend is `https://api.example.com`. Your backend server must serve `OPTIONS` request with `allow-control-allow-origin` echoing the value of `origin`.

```golang
var matcher *originmatcher.T

func init() {
	matcher, _ = originmatcher.Parse("https://www.example.com,https://example.com")
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		origin := r.Header.Get("origin")
		if matcher.MatchOrigin(origin) {
			w.Header().Set("allow-control-allow-origin", origin)
			// Set more CORS headers as you like
		} else {
			// Unknown cross-origin request!
		}
	} else {
		// Serve your request as normal
	}
}
```
