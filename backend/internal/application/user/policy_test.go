package user

import (
	"strings"
	"testing"
)

func TestNormalizeUsernamePolicy(t *testing.T) {
	if got, err := NormalizeUsername(" Alice_01 "); err != nil || got != "alice_01" {
		t.Fatalf("expected normalized username, got %q err=%v", got, err)
	}

	for _, raw := range []string{"ab", "abcdefghijklmnopq", "admin", "user@example.com", "-alice", "alice.", "alice_"} {
		if _, err := NormalizeUsername(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func TestNormalizeDisplayNamePolicy(t *testing.T) {
	if got, err := NormalizeDisplayName(" Chenyme "); err != nil || got != "Chenyme" {
		t.Fatalf("expected normalized display name, got %q err=%v", got, err)
	}

	for _, raw := range []string{"ab", "abcdefghijklmnopq"} {
		if _, err := NormalizeDisplayName(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func TestNormalizePasswordPolicy(t *testing.T) {
	if got, err := NormalizePassword(" deeix2026 "); err != nil || got != "deeix2026" {
		t.Fatalf("expected normalized password, got %q err=%v", got, err)
	}

	for _, raw := range []string{"short7", "12345678"} {
		if _, err := NormalizePassword(raw); err == nil {
			t.Fatalf("expected password %q to be rejected", raw)
		}
	}
}

func TestNormalizeProfilePreferencesPolicy(t *testing.T) {
	if got, err := NormalizeProfilePreferences("  answer concisely  "); err != nil || got != "answer concisely" {
		t.Fatalf("expected normalized profile preferences, got %q err=%v", got, err)
	}

	if got, err := NormalizeProfilePreferences(strings.Repeat("中", ProfilePreferencesMaxLength)); err != nil || len([]rune(got)) != ProfilePreferencesMaxLength {
		t.Fatalf("expected %d characters to be accepted, got %d err=%v", ProfilePreferencesMaxLength, len([]rune(got)), err)
	}

	if _, err := NormalizeProfilePreferences(strings.Repeat("中", ProfilePreferencesMaxLength+1)); err == nil {
		t.Fatalf("expected %d characters to be rejected", ProfilePreferencesMaxLength+1)
	}
}
