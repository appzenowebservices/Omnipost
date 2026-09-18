package push

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// tokenSourceFromKey builds an OAuth2 token source from a service-account
// JSON key for the FCM scope.
func tokenSourceFromKey(ctx context.Context, key []byte) (oauth2.TokenSource, error) {
	cfg, err := google.JWTConfigFromJSON(key, fcmScope)
	if err != nil {
		return nil, fmt.Errorf("invalid service-account key: %v", err)
	}
	return cfg.TokenSource(ctx), nil
}

// authTransport injects a Bearer token into every request.
type authTransport struct {
	base http.RoundTripper
	src  oauth2.TokenSource
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := t.src.Token()
	if err != nil {
		return nil, fmt.Errorf("fcm auth failed: %v", err)
	}

	out := req.Clone(req.Context())
	tok.SetAuthHeader(out)

	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(out)
}
