package push

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// tokenSourceFromKey builds an OAuth2 token source from a service-account
// JSON key for the FCM scope, using a self-signed JWT bearer assertion.
// Implemented with the standard library so no extra modules are required.
func tokenSourceFromKey(_ context.Context, key []byte) (oauth2.TokenSource, error) {
	var sa struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
	}
	if err := json.Unmarshal(key, &sa); err != nil {
		return nil, fmt.Errorf("invalid service-account key: %v", err)
	}
	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, fmt.Errorf("invalid service-account key: missing client_email or private_key")
	}

	priv, err := parsePrivateKey(sa.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid service-account private_key: %v", err)
	}

	src := &jwtSource{
		clientEmail: sa.ClientEmail,
		privateKey:  priv,
		http:        &http.Client{Timeout: 15 * time.Second},
	}
	return oauth2.ReuseTokenSource(nil, src), nil
}

func parsePrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := k.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, fmt.Errorf("private key is not RSA")
	}

	// Fallback for legacy PKCS#1 ("RSA PRIVATE KEY") blocks.
	if rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return rsaKey, nil
	} else {
		return nil, fmt.Errorf("cannot parse private key: %v", err)
	}
}

// jwtSource mints OAuth2 access tokens via the JWT bearer grant.
type jwtSource struct {
	clientEmail string
	privateKey  *rsa.PrivateKey
	http        *http.Client
}

func (s *jwtSource) Token() (*oauth2.Token, error) {
	now := time.Now()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))

	claims, err := json.Marshal(map[string]any{
		"iss":   s.clientEmail,
		"scope": fcmScope,
		"aud":   "https://oauth2.googleapis.com/token",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	})
	if err != nil {
		return nil, err
	}
	payload := base64.RawURLEncoding.EncodeToString(claims)

	signingInput := header + "." + payload
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign JWT assertion: %v", err)
	}
	assertion := signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)

	req, err := http.NewRequest(http.MethodPost,
		"https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token exchange failed: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("token exchange failed: %v", err)
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("token exchange failed: empty access_token")
	}

	expiry := now.Add(time.Duration(out.ExpiresIn) * time.Second)
	if out.ExpiresIn <= 0 {
		expiry = now.Add(time.Hour)
	} else {
		// Refresh a minute early.
		expiry = expiry.Add(-time.Minute)
	}

	// Empty TokenType defaults to "Bearer" when the auth header is set.
	return &oauth2.Token{AccessToken: out.AccessToken, Expiry: expiry}, nil
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
