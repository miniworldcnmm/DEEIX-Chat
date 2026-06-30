package conversation

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

// mergeStubRepo 嵌入 ConversationRepository 接口，只覆盖 GetUserSettingValue，用于 mergeUserModelDefaults 测试。
type mergeStubRepo struct {
	repository.ConversationRepository
	settings map[uint]map[string]string
}

func (r *mergeStubRepo) GetUserSettingValue(_ context.Context, userID uint, key string) (string, error) {
	if m, ok := r.settings[userID]; ok {
		return m[key], nil
	}
	return "", nil
}

func newMergeTestService() *Service {
	return &Service{
		repo: &mergeStubRepo{settings: make(map[uint]map[string]string)},
	}
}

func (svc *Service) seed(userID uint, key, value string) {
	if stub, ok := svc.repo.(*mergeStubRepo); ok {
		if _, ok := stub.settings[userID]; !ok {
			stub.settings[userID] = make(map[string]string)
		}
		stub.settings[userID][key] = value
	}
}

func TestStripThinkingFieldsRemovesAllKnownPaths(t *testing.T) {
	options := map[string]interface{}{
		"reasoning_effort": "high",
		"reasoning":        map[string]interface{}{"effort": "high"},
		"thinking":         map[string]interface{}{"type": "enabled"},
		"thinkingConfig":   map[string]interface{}{"thinkingLevel": "high"},
		"effort":           "high",
		"enable_thinking":  true,
		"thinking_budget":  1024,
		"budget_tokens":    1024,
		"thinking_level":   "high",
		"thinkingLevel":    "high",
		"generationConfig": map[string]interface{}{"thinkingConfig": map[string]interface{}{"thinkingLevel": "high"}, "temperature": 0.5},
	}
	stripThinkingFields(options)
	for _, key := range []string{"reasoning_effort", "reasoning", "thinking", "thinkingConfig", "effort", "enable_thinking", "thinking_budget", "budget_tokens", "thinking_level", "thinkingLevel"} {
		if _, ok := options[key]; ok {
			t.Fatalf("expected %q removed, still present", key)
		}
	}
	if gc, ok := options["generationConfig"].(map[string]interface{}); ok {
		if _, present := gc["thinkingConfig"]; present {
			t.Fatalf("expected generationConfig.thinkingConfig removed")
		}
		if _, present := gc["temperature"]; !present {
			t.Fatalf("expected generationConfig.temperature preserved")
		}
	} else {
		t.Fatalf("expected generationConfig preserved")
	}
}

func TestProtocolThinkingEffortPathMatchesProtocol(t *testing.T) {
	cases := map[string]string{
		"openai":     "reasoning.effort",
		"openrouter": "reasoning.effort",
		"anthropic":  "",
		"google":     "generationConfig.thinkingConfig.thinkingLevel",
		"":           "reasoning.effort",
	}
	for protocol, expected := range cases {
		got := protocolThinkingEffortPath(protocol)
		if got != expected {
			t.Fatalf("protocol %q: expected %q, got %q", protocol, expected, got)
		}
	}
}

func TestMergeUserModelDefaultsGlobalReasoningEffortInjectedWhenExplicitAbsent(t *testing.T) {
	svc := newMergeTestService()
	svc.seed(1, "chat.default_thinking_enabled", "true")
	svc.seed(1, "chat.default_temperature", "1")
	svc.seed(1, "chat.default_reasoning_effort", "high")
	result := svc.mergeUserModelDefaults(context.Background(), 1, "gpt-test", nil, "openai")
	if got := readNested(result, []string{"reasoning", "effort"}); got != "high" {
		t.Fatalf("expected reasoning.effort=high, got %v", got)
	}
	if temp, ok := result["temperature"].(float64); !ok || temp != 1 {
		t.Fatalf("expected temperature=1, got %v", result["temperature"])
	}
}

func TestMergeUserModelDefaultsExplicitOverridesGlobal(t *testing.T) {
	svc := newMergeTestService()
	svc.seed(1, "chat.default_thinking_enabled", "true")
	svc.seed(1, "chat.default_temperature", "1")
	svc.seed(1, "chat.default_reasoning_effort", "high")
	explicit := map[string]interface{}{
		"reasoning_effort": "low",
		"temperature":      0.3,
	}
	result := svc.mergeUserModelDefaults(context.Background(), 1, "gpt-test", explicit, "openai_chat_completions")
	if got, ok := result["reasoning_effort"].(string); !ok || got != "low" {
		t.Fatalf("expected reasoning_effort=low preserved, got %v", result["reasoning_effort"])
	}
	if got, ok := result["temperature"].(float64); !ok || got != 0.3 {
		t.Fatalf("expected temperature=0.3 preserved, got %v", result["temperature"])
	}
	if _, ok := readNested(result, []string{"reasoning", "effort"}).(string); ok {
		t.Fatalf("expected reasoning.effort NOT injected when explicit reasoning_effort set")
	}
}

func TestMergeUserModelDefaultsSingleModelOverridesGlobal(t *testing.T) {
	svc := newMergeTestService()
	svc.seed(1, "chat.default_thinking_enabled", "true")
	svc.seed(1, "chat.default_temperature", "1")
	svc.seed(1, "chat.default_reasoning_effort", "high")
	temp := 0.7
	effort := "xhigh"
	enabled := false
	payload := map[string]interface{}{
		"thinking_enabled": enabled,
		"temperature":      temp,
		"reasoning_effort": effort,
	}
	payloadJSON, _ := json.Marshal(payload)
	svc.seed(1, "chat.model_option:gpt-test", string(payloadJSON))
	result := svc.mergeUserModelDefaults(context.Background(), 1, "gpt-test", nil, "openai")
	if got, ok := result["temperature"].(float64); !ok || got != 0.7 {
		t.Fatalf("expected single model temperature=0.7, got %v", result["temperature"])
	}
	// 思考开关被单模型配置关闭，所以所有思考字段都应被清除。
	if _, ok := result["reasoning_effort"]; ok {
		t.Fatalf("expected reasoning_effort removed because thinking disabled")
	}
	if _, ok := result["reasoning"]; ok {
		t.Fatalf("expected reasoning removed because thinking disabled")
	}
	if _, ok := result["temperature"]; !ok {
		t.Fatalf("expected temperature preserved even when thinking disabled")
	}
}

func TestMergeUserModelDefaultsGlobalThinkingDisabledStripsAll(t *testing.T) {
	svc := newMergeTestService()
	svc.seed(1, "chat.default_thinking_enabled", "false")
	svc.seed(1, "chat.default_temperature", "1")
	svc.seed(1, "chat.default_reasoning_effort", "high")
	result := svc.mergeUserModelDefaults(context.Background(), 1, "gpt-test", map[string]interface{}{
		"reasoning_effort": "medium",
	}, "openai_chat_completions")
	if _, ok := result["reasoning_effort"]; ok {
		t.Fatalf("expected reasoning_effort stripped when global thinking disabled")
	}
	if _, ok := result["reasoning"]; ok {
		t.Fatalf("expected reasoning stripped when global thinking disabled")
	}
}

func TestMergeUserModelDefaultsExplicitThinkingEnabledOverridesGlobalDisabled(t *testing.T) {
	svc := newMergeTestService()
	svc.seed(1, "chat.default_thinking_enabled", "false")
	svc.seed(1, "chat.default_reasoning_effort", "high")
	explicit := map[string]interface{}{
		"enable_thinking":  true,
		"reasoning_effort": "max",
	}
	result := svc.mergeUserModelDefaults(context.Background(), 1, "gpt-test", explicit, "anthropic_messages")
	if got, ok := result["reasoning_effort"].(string); !ok || got != "max" {
		t.Fatalf("expected explicit reasoning_effort=max preserved, got %v", result["reasoning_effort"])
	}
	if got, ok := result["enable_thinking"].(bool); !ok || !got {
		t.Fatalf("expected explicit enable_thinking=true preserved, got %v", result["enable_thinking"])
	}
}

func TestMergeUserModelDefaultsRemovesUnsupportedThinkingToggleBeforeUpstream(t *testing.T) {
	svc := newMergeTestService()
	svc.seed(1, "chat.default_thinking_enabled", "false")
	svc.seed(1, "chat.default_reasoning_effort", "high")
	result := svc.mergeUserModelDefaults(context.Background(), 1, "gpt-test", map[string]interface{}{
		"enable_thinking":  true,
		"reasoning_effort": "medium",
	}, "openai_chat_completions")
	if _, ok := result["enable_thinking"]; ok {
		t.Fatalf("expected unsupported enable_thinking control field removed before filtering")
	}
	if got, ok := result["reasoning_effort"].(string); !ok || got != "medium" {
		t.Fatalf("expected explicit reasoning effort preserved, got %v", result["reasoning_effort"])
	}
}

func TestMergeUserModelDefaultsGlobalDefaultEffortEmptyDoesNotInject(t *testing.T) {
	svc := newMergeTestService()
	svc.seed(1, "chat.default_thinking_enabled", "true")
	svc.seed(1, "chat.default_temperature", "1.5")
	svc.seed(1, "chat.default_reasoning_effort", "")
	result := svc.mergeUserModelDefaults(context.Background(), 1, "gpt-test", nil, "openai")
	if readNested(result, []string{"reasoning", "effort"}) != nil {
		t.Fatalf("expected reasoning.effort NOT injected when global default empty")
	}
	if got, ok := result["temperature"].(float64); !ok || got != 1.5 {
		t.Fatalf("expected temperature=1.5, got %v", result["temperature"])
	}
}

func TestMergeUserModelDefaultsGeminiUsesNestedTemperature(t *testing.T) {
	svc := newMergeTestService()
	svc.seed(1, "chat.default_thinking_enabled", "true")
	svc.seed(1, "chat.default_temperature", "0.9")
	svc.seed(1, "chat.default_reasoning_effort", "high")
	result := svc.mergeUserModelDefaults(context.Background(), 1, "gemini-test", nil, "google")
	gc, ok := result["generationConfig"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected generationConfig present for gemini")
	}
	if got, ok := gc["temperature"].(float64); !ok || got != 0.9 {
		t.Fatalf("expected generationConfig.temperature=0.9, got %v", gc["temperature"])
	}
	if got, ok := gc["thinkingConfig"].(map[string]interface{}); !ok {
		t.Fatalf("expected generationConfig.thinkingConfig present")
	} else if got["thinkingLevel"] != "high" {
		t.Fatalf("expected thinkingConfig.thinkingLevel=high, got %v", got["thinkingLevel"])
	}
}

func TestMergeUserModelDefaultsEmptyPlatformModelReturnsClone(t *testing.T) {
	svc := newMergeTestService()
	explicit := map[string]interface{}{"temperature": 0.5}
	result := svc.mergeUserModelDefaults(context.Background(), 1, "", explicit, "openai")
	if got, ok := result["temperature"].(float64); !ok || got != 0.5 {
		t.Fatalf("expected explicit preserved when platform model empty, got %v", result["temperature"])
	}
}

func readNested(options map[string]interface{}, path []string) interface{} {
	current := options
	for i, segment := range path {
		if i == len(path)-1 {
			return current[segment]
		}
		next, ok := current[segment].(map[string]interface{})
		if !ok {
			return nil
		}
		current = next
	}
	return nil
}
