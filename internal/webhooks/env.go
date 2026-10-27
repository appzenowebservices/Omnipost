package webhooks

import (
	"os"
	"strings"
)

// envRule is one PATRA_CONFIRM_WEBHOOKS entry.
type envRule struct {
	ref    string
	url    string
	secret string
}

// parseEnvRules parses "<list-ref>|<url>|<secret>" entries separated by ';'.
// The list ref may be a list UUID or its exact name. Malformed entries are
// skipped.
func parseEnvRules(raw string) []envRule {
	var out []envRule
	for _, chunk := range strings.Split(raw, ";") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		parts := strings.SplitN(chunk, "|", 3)
		if len(parts) != 3 {
			continue
		}
		ref := strings.TrimSpace(parts[0])
		u := strings.TrimSpace(parts[1])
		if ref == "" || u == "" {
			continue
		}
		out = append(out, envRule{ref: ref, url: u, secret: strings.TrimSpace(parts[2])})
	}
	return out
}

// EnvTarget resolves the environment-configured webhook for a list.
//
// PATRA_CONFIRM_WEBHOOKS is a ';'-separated list of
// "<list-uuid-or-name>|<url>|<secret>" rules, one per list, so every list can
// have its own endpoint and HMAC secret. PATRA_CONFIRM_WEBHOOK_URL (with
// PATRA_CONFIRM_WEBHOOK_SECRET) is an optional catch-all for lists without a
// rule.
//
// configured reports whether any env webhook config exists at all; when it is
// true callers must not fall back to database targets. ok reports whether this
// particular list has a target.
func EnvTarget(uuid, name string) (url, secret string, configured, ok bool) {
	raw := strings.TrimSpace(os.Getenv("PATRA_CONFIRM_WEBHOOKS"))
	catchURL := strings.TrimSpace(os.Getenv("PATRA_CONFIRM_WEBHOOK_URL"))
	configured = raw != "" || catchURL != ""

	for _, r := range parseEnvRules(raw) {
		if strings.EqualFold(r.ref, uuid) || strings.EqualFold(r.ref, name) {
			return r.url, r.secret, configured, true
		}
	}
	if catchURL != "" {
		return catchURL, strings.TrimSpace(os.Getenv("PATRA_CONFIRM_WEBHOOK_SECRET")), configured, true
	}
	return "", "", configured, false
}
