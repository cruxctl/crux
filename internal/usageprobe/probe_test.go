package usageprobe

import (
	"strings"
	"testing"
)

func TestSanitizeRedactsStructuredSensitiveFields(t *testing.T) {
	input := `{
  "loggedIn": true,
  "email": "user@example.com",
  "orgId": "ff5592f1-f392-4765-a986-e8f41894c79f",
  "orgName": "user@example.com's Organization",
  "subscriptionType": "max"
}`

	got := Sanitize(input)
	for _, forbidden := range []string{"user@example.com", "ff5592f1-f392-4765-a986-e8f41894c79f", "Organization"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("sanitize leaked %q in %s", forbidden, got)
		}
	}
	for _, want := range []string{`"loggedIn": true`, `"subscriptionType": "max"`, `"<redacted>"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("sanitize output missing %q: %s", want, got)
		}
	}
}

func TestSanitizeRedactsPlainTextEmail(t *testing.T) {
	got := Sanitize("logged in as user@example.com")
	if strings.Contains(got, "user@example.com") {
		t.Fatalf("sanitize leaked email: %s", got)
	}
	if !strings.Contains(got, "<redacted-email>") {
		t.Fatalf("sanitize did not mark redacted email: %s", got)
	}
}

func TestSanitizeRedactsPlainTextUUID(t *testing.T) {
	got := Sanitize("session [5ce1e9c3-1a19-40ac-a9ad-05caf4a58e0f]")
	if strings.Contains(got, "5ce1e9c3-1a19-40ac-a9ad-05caf4a58e0f") {
		t.Fatalf("sanitize leaked UUID: %s", got)
	}
	if !strings.Contains(got, "<redacted-id>") {
		t.Fatalf("sanitize did not mark redacted UUID: %s", got)
	}
}
