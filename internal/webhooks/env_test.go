package webhooks

import "testing"

func TestParseEnvRules(t *testing.T) {
	rules := parseEnvRules(
		" 8e53ee6a-5876-41dd-9156-e9dc90095b88 | https://one.test/hook | s1 ; " +
			"apnidesidukaan-vendors|https://two.test/hook|s2;" +
			"broken;" +
			"|https://missing-ref.test/hook|s3;",
	)

	if len(rules) != 2 {
		t.Fatalf("expected 2 valid rules, got %d: %#v", len(rules), rules)
	}
	if rules[0].ref != "8e53ee6a-5876-41dd-9156-e9dc90095b88" ||
		rules[0].url != "https://one.test/hook" || rules[0].secret != "s1" {
		t.Fatalf("unexpected first rule: %#v", rules[0])
	}
	if rules[1].ref != "apnidesidukaan-vendors" ||
		rules[1].url != "https://two.test/hook" || rules[1].secret != "s2" {
		t.Fatalf("unexpected second rule: %#v", rules[1])
	}
}

func TestEnvTarget(t *testing.T) {
	t.Setenv("PATRA_CONFIRM_WEBHOOKS",
		"8e53ee6a-5876-41dd-9156-e9dc90095b88|https://one.test/hook|s1;"+
			"apnidesidukaan-vendors|https://two.test/hook|s2")
	t.Setenv("PATRA_CONFIRM_WEBHOOK_URL", "")
	t.Setenv("PATRA_CONFIRM_WEBHOOK_SECRET", "")

	// Match by UUID.
	u, s, configured, ok := EnvTarget("8e53ee6a-5876-41dd-9156-e9dc90095b88", "apnidesidukaan")
	if !configured || !ok || u != "https://one.test/hook" || s != "s1" {
		t.Fatalf("uuid match failed: configured=%v ok=%v url=%q secret=%q", configured, ok, u, s)
	}

	// Match by exact name (case-insensitive).
	u, s, configured, ok = EnvTarget("f325d481-7cf1-446a-bb0a-a81fab4b7fc2", "ApniDesiDukaan-Vendors")
	if !configured || !ok || u != "https://two.test/hook" || s != "s2" {
		t.Fatalf("name match failed: configured=%v ok=%v url=%q secret=%q", configured, ok, u, s)
	}

	// No rule and no catch-all: configured, but this list has no target.
	_, _, configured, ok = EnvTarget("other", "other")
	if !configured || ok {
		t.Fatalf("expected configured without a target: configured=%v ok=%v", configured, ok)
	}

	// Catch-all applies to unmatched lists.
	t.Setenv("PATRA_CONFIRM_WEBHOOK_URL", "https://catch-all.test/hook")
	t.Setenv("PATRA_CONFIRM_WEBHOOK_SECRET", "catch-secret")
	u, s, configured, ok = EnvTarget("other", "other")
	if !configured || !ok || u != "https://catch-all.test/hook" || s != "catch-secret" {
		t.Fatalf("catch-all failed: configured=%v ok=%v url=%q secret=%q", configured, ok, u, s)
	}
}

func TestEnvTargetDisabled(t *testing.T) {
	t.Setenv("PATRA_CONFIRM_WEBHOOKS", "")
	t.Setenv("PATRA_CONFIRM_WEBHOOK_URL", "")

	u, s, configured, ok := EnvTarget("uuid", "name")
	if configured || ok || u != "" || s != "" {
		t.Fatalf("expected disabled: configured=%v ok=%v url=%q secret=%q", configured, ok, u, s)
	}
}
