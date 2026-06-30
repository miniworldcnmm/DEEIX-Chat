package auth

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin/binding"
)

func TestPatchMeRequestProfilePreferencesLimit(t *testing.T) {
	accepted := strings.Repeat("中", 15000)
	if err := binding.Validator.ValidateStruct(PatchMeRequest{ProfilePreferences: &accepted}); err != nil {
		t.Fatalf("expected 15000 characters to be accepted, got %v", err)
	}

	rejected := strings.Repeat("中", 15001)
	if err := binding.Validator.ValidateStruct(PatchMeRequest{ProfilePreferences: &rejected}); err == nil {
		t.Fatal("expected 15001 characters to be rejected")
	}
}
