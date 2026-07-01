package usersettings

import (
	"context"
	"testing"

	domainusersettings "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/usersettings"
)

// stubRepo 仅用于校验/读写测试。
type stubRepo struct {
	values map[uint]map[string]string
}

func newStubRepo() *stubRepo {
	return &stubRepo{values: make(map[uint]map[string]string)}
}

func (r *stubRepo) ListByUserID(_ context.Context, userID uint) ([]domainusersettings.UserSetting, error) {
	m, ok := r.values[userID]
	if !ok {
		return nil, nil
	}
	result := make([]domainusersettings.UserSetting, 0, len(m))
	for k, v := range m {
		result = append(result, domainusersettings.UserSetting{UserID: userID, Key: k, Value: v})
	}
	return result, nil
}

func (r *stubRepo) Upsert(_ context.Context, items []domainusersettings.UserSetting) error {
	for _, it := range items {
		if r.values[it.UserID] == nil {
			r.values[it.UserID] = make(map[string]string)
		}
		r.values[it.UserID][it.Key] = it.Value
	}
	return nil
}

func (r *stubRepo) Delete(_ context.Context, userID uint, key string) error {
	if m, ok := r.values[userID]; ok {
		delete(m, key)
	}
	return nil
}

func TestValidateDefaultMCPToolIDs(t *testing.T) {
	t.Parallel()

	validValues := []string{
		"[]",
		"[1]",
		"[1,2,3]",
		" [42] ",
	}
	for _, value := range validValues {
		if err := validateDefaultMCPToolIDs(value, "chat.default_mcp_tool_ids"); err != nil {
			t.Fatalf("expected %s to be valid, got %v", value, err)
		}
	}

	invalidValues := []string{
		"",
		"{}",
		"[0]",
		"[-1]",
		"[1.5]",
		`["1"]`,
	}
	for _, value := range invalidValues {
		if err := validateDefaultMCPToolIDs(value, "chat.default_mcp_tool_ids"); err == nil {
			t.Fatalf("expected %s to be invalid", value)
		}
	}
}

func TestDefaultMCPToolIDsSettingIsAllowed(t *testing.T) {
	t.Parallel()

	if got := allowedKeys["chat.default_mcp_tool_ids"]; got != "[]" {
		t.Fatalf("expected chat.default_mcp_tool_ids default to be [], got %q", got)
	}
	if err := validateValue("chat.default_mcp_tool_ids", "[1,2,3]"); err != nil {
		t.Fatalf("expected chat.default_mcp_tool_ids to be accepted, got %v", err)
	}
}

func TestContentWidthSettingIsAllowed(t *testing.T) {
	t.Parallel()

	if got := allowedKeys["chat.content_width"]; got != "compact" {
		t.Fatalf("expected chat.content_width default to be compact, got %q", got)
	}
	for _, value := range []string{"compact", "standard", "wide"} {
		if err := validateValue("chat.content_width", value); err != nil {
			t.Fatalf("expected chat.content_width=%s to be accepted, got %v", value, err)
		}
	}
	if err := validateValue("chat.content_width", "loose"); err == nil {
		t.Fatal("expected invalid chat.content_width to be rejected")
	}
}

func TestReuseModelOptionsSettingIsAllowed(t *testing.T) {
	t.Parallel()

	if got := allowedKeys["chat.reuse_model_options"]; got != "true" {
		t.Fatalf("expected chat.reuse_model_options default to be true, got %q", got)
	}
	for _, value := range []string{"true", "false"} {
		if err := validateValue("chat.reuse_model_options", value); err != nil {
			t.Fatalf("expected chat.reuse_model_options=%s to be accepted, got %v", value, err)
		}
	}
	if err := validateValue("chat.reuse_model_options", "yes"); err == nil {
		t.Fatal("expected invalid chat.reuse_model_options to be rejected")
	}
}

func TestValidateDefaultTemperatureAcceptsBounds(t *testing.T) {
	t.Parallel()
	if err := validateDefaultTemperature("0", "chat.default_temperature"); err != nil {
		t.Fatalf("0 should pass: %v", err)
	}
	if err := validateDefaultTemperature("2", "chat.default_temperature"); err != nil {
		t.Fatalf("2 should pass: %v", err)
	}
	if err := validateDefaultTemperature("1.5", "chat.default_temperature"); err != nil {
		t.Fatalf("1.5 should pass: %v", err)
	}
}

func TestValidateDefaultTemperatureRejectsOutOfRange(t *testing.T) {
	t.Parallel()
	if err := validateDefaultTemperature("2.1", "chat.default_temperature"); err == nil {
		t.Fatal("2.1 should fail")
	}
	if err := validateDefaultTemperature("-0.1", "chat.default_temperature"); err == nil {
		t.Fatal("-0.1 should fail")
	}
	if err := validateDefaultTemperature("abc", "chat.default_temperature"); err == nil {
		t.Fatal("abc should fail")
	}
	if err := validateDefaultTemperature("", "chat.default_temperature"); err == nil {
		t.Fatal("empty should fail")
	}
}

func TestDefaultThinkingEnabledSettingIsAllowed(t *testing.T) {
	t.Parallel()
	if got := allowedKeys["chat.default_thinking_enabled"]; got != "true" {
		t.Fatalf("expected chat.default_thinking_enabled default true, got %q", got)
	}
	for _, value := range []string{"true", "false"} {
		if err := validateValue("chat.default_thinking_enabled", value); err != nil {
			t.Fatalf("expected %s accepted, got %v", value, err)
		}
	}
	if err := validateValue("chat.default_thinking_enabled", "yes"); err == nil {
		t.Fatal("expected invalid value rejected")
	}
}

func TestMemoryEnabledSettingDefaultsToFalse(t *testing.T) {
	t.Parallel()
	if got := allowedKeys["chat.memory_enabled"]; got != "false" {
		t.Fatalf("expected chat.memory_enabled default false, got %q", got)
	}
	for _, value := range []string{"true", "false"} {
		if err := validateValue("chat.memory_enabled", value); err != nil {
			t.Fatalf("expected %s accepted, got %v", value, err)
		}
	}
	if err := validateValue("chat.memory_enabled", "yes"); err == nil {
		t.Fatal("expected invalid value rejected")
	}
}

func TestValidateModelOptionPayloadAcceptsKnownFields(t *testing.T) {
	t.Parallel()
	value := `{"thinking_enabled":true,"temperature":1.0,"reasoning_effort":"high"}`
	if err := validateModelOptionPayload("chat.model_option:gpt-test", value); err != nil {
		t.Fatalf("expected pass, got %v", err)
	}
}

func TestValidateModelOptionPayloadRejectsUnknownField(t *testing.T) {
	t.Parallel()
	if err := validateModelOptionPayload("chat.model_option:gpt-test", `{"top_p":0.9}`); err == nil {
		t.Fatal("expected unknown field rejected")
	}
}

func TestValidateModelOptionPayloadRejectsBadType(t *testing.T) {
	t.Parallel()
	if err := validateModelOptionPayload("chat.model_option:gpt-test", `{"thinking_enabled":"yes"}`); err == nil {
		t.Fatal("expected bool type check")
	}
	if err := validateModelOptionPayload("chat.model_option:gpt-test", `{"temperature":"hot"}`); err == nil {
		t.Fatal("expected number type check")
	}
	if err := validateModelOptionPayload("chat.model_option:gpt-test", `{"reasoning_effort":5}`); err == nil {
		t.Fatal("expected string type check")
	}
	if err := validateModelOptionPayload("chat.model_option:gpt-test", `{"temperature":3}`); err == nil {
		t.Fatal("expected temperature range check")
	}
}

func TestValidateModelOptionPayloadRejectsEmptyName(t *testing.T) {
	t.Parallel()
	if err := validateModelOptionPayload("chat.model_option:", `{"temperature":1}`); err == nil {
		t.Fatal("expected empty name rejected")
	}
}

func TestUpsertUserModelOptionRejectsInvalidName(t *testing.T) {
	svc := NewService(newStubRepo())
	if err := svc.UpsertUserModelOption(context.Background(), 1, "  ", ModelOptionPayload{}); err == nil {
		t.Fatal("expected empty name rejected")
	}
}

func TestUpsertAndDeleteUserModelOptionRoundTrip(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	temp := 0.5
	effort := "high"
	enabled := true
	if err := svc.UpsertUserModelOption(context.Background(), 1, "gpt-test", ModelOptionPayload{
		Temperature:     &temp,
		ReasoningEffort: &effort,
		ThinkingEnabled: &enabled,
	}); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	options, err := svc.ListUserModelOptions(context.Background(), 1)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	payload, ok := options["gpt-test"]
	if !ok {
		t.Fatal("expected gpt-test payload present")
	}
	if payload.Temperature == nil || *payload.Temperature != 0.5 {
		t.Fatalf("expected temperature=0.5, got %v", payload.Temperature)
	}
	if payload.ReasoningEffort == nil || *payload.ReasoningEffort != "high" {
		t.Fatalf("expected reasoning_effort=high, got %v", payload.ReasoningEffort)
	}
	if payload.ThinkingEnabled == nil || !*payload.ThinkingEnabled {
		t.Fatalf("expected thinking_enabled=true, got %v", payload.ThinkingEnabled)
	}
	if err := svc.DeleteUserModelOption(context.Background(), 1, "gpt-test"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	options, _ = svc.ListUserModelOptions(context.Background(), 1)
	if _, ok := options["gpt-test"]; ok {
		t.Fatal("expected gpt-test removed after delete")
	}
}
