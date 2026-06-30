package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	appusersettings "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/usersettings"
)

// 用户级模型默认配置 key。
const (
	userSettingKeyDefaultThinkingEnabled = "chat.default_thinking_enabled"
	userSettingKeyDefaultTemperature     = "chat.default_temperature"
	userSettingKeyDefaultReasoningEffort = "chat.default_reasoning_effort"
)

// thinkingFieldPaths 汇总各协议可能出现的思考相关字段路径，用于开关关闭时联合清除。
var thinkingFieldPaths = []string{
	"reasoning_effort",
	"reasoning",
	"thinking",
	"thinkingConfig",
	"effort",
	"enable_thinking",
	"thinking_budget",
	"thinking_budget_tokens",
	"budget_tokens",
	"thinking_level",
	"thinkingLevel",
}

// thinkingNestedPaths 是嵌套在 generationConfig 下的思考子字段。
var thinkingNestedPaths = [][]string{
	{"generationConfig", "thinkingConfig"},
}

// protocolThinkingEffortPath 返回当前协议的思考等级注入路径，空表示不注入。
func protocolThinkingEffortPath(protocol string) string {
	switch modelOptionPolicyProtocolKey(protocol) {
	case "openai_chat_completions", "openrouter_chat_completions":
		return "reasoning_effort"
	case "openai_responses", "openrouter_responses", "xai_responses":
		return "reasoning.effort"
	case "gemini_generate_content":
		return "generationConfig.thinkingConfig.thinkingLevel"
	default:
		return ""
	}
}

// protocolThinkingEnabledPath 返回当前协议的思考开关注入路径，空表示不注入（用 strip 语义即可）。
func protocolThinkingEnabledPath(protocol string) string {
	switch modelOptionPolicyProtocolKey(protocol) {
	case "anthropic_messages":
		return "enable_thinking"
	default:
		return ""
	}
}

// hasExplicitThinkingEnabled 判断 explicit options 是否显式表达了思考开关状态。
// 返回 (enabled, ok)：ok=true 表示有显式表达。
func hasExplicitThinkingEnabled(options map[string]interface{}) (bool, bool) {
	if v, ok := options["enable_thinking"].(bool); ok {
		return v, true
	}
	if thinking, ok := options["thinking"].(map[string]interface{}); ok {
		if rawType, ok := thinking["type"].(string); ok {
			switch strings.ToLower(strings.TrimSpace(rawType)) {
			case "enabled", "adaptive":
				return true, true
			case "disabled":
				return false, true
			}
		}
	}
	return false, false
}

// hasExplicitReasoningEffort 判断 explicit options 是否显式表达了思考等级。
func hasExplicitReasoningEffort(options map[string]interface{}) bool {
	for _, key := range []string{"reasoning_effort", "effort"} {
		if v, ok := options[key]; ok && v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return true
			}
		}
	}
	if reasoning, ok := options["reasoning"].(map[string]interface{}); ok {
		if v, ok := reasoning["effort"].(string); ok && strings.TrimSpace(v) != "" {
			return true
		}
	}
	if generationConfig, ok := options["generationConfig"].(map[string]interface{}); ok {
		if thinkingConfig, ok := generationConfig["thinkingConfig"].(map[string]interface{}); ok {
			if v, ok := thinkingConfig["thinkingLevel"].(string); ok && strings.TrimSpace(v) != "" {
				return true
			}
		}
	}
	return false
}

// hasExplicitTemperature 判断 explicit options 是否显式设置了温度。
func hasExplicitTemperature(options map[string]interface{}) bool {
	if v, ok := options["temperature"]; ok && v != nil {
		return true
	}
	if generationConfig, ok := options["generationConfig"].(map[string]interface{}); ok {
		if v, ok := generationConfig["temperature"]; ok && v != nil {
			return true
		}
	}
	return false
}

// stripThinkingFields 清除 options 中所有思考相关字段。
func stripThinkingFields(options map[string]interface{}) {
	for _, key := range thinkingFieldPaths {
		delete(options, key)
	}
	for _, path := range thinkingNestedPaths {
		deleteNestedPath(options, path)
	}
	if generationConfig, ok := options["generationConfig"].(map[string]interface{}); ok {
		delete(generationConfig, "thinkingConfig")
		if len(generationConfig) == 0 {
			delete(options, "generationConfig")
		}
	}
}

func deleteNestedPath(options map[string]interface{}, path []string) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		delete(options, path[0])
		return
	}
	nested, ok := options[path[0]].(map[string]interface{})
	if !ok {
		return
	}
	deleteNestedPath(nested, path[1:])
	if len(nested) == 0 {
		delete(options, path[0])
	}
}

// setNestedPath 写入嵌套路径。
func setNestedPath(options map[string]interface{}, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		options[path[0]] = value
		return
	}
	nested, ok := options[path[0]].(map[string]interface{})
	if !ok {
		nested = make(map[string]interface{})
		options[path[0]] = nested
	}
	setNestedPath(nested, path[1:], value)
}

// injectThinkingEnabled 按协议注入思考开关字段。
func injectThinkingEnabled(options map[string]interface{}, protocol string, enabled bool) {
	path := protocolThinkingEnabledPath(protocol)
	if path == "" {
		return
	}
	if enabled {
		options[path] = true
	} else {
		options[path] = false
	}
}

// injectReasoningEffort 按协议注入思考等级字段。effort 为空字符串时不注入。
func injectReasoningEffort(options map[string]interface{}, protocol string, effort string) {
	effort = strings.TrimSpace(effort)
	if effort == "" {
		return
	}
	path := protocolThinkingEffortPath(protocol)
	if path == "" {
		return
	}
	setNestedPath(options, strings.Split(path, "."), effort)
}

// injectTemperature 注入温度字段。openai 协议族用顶层 temperature，gemini 用 generationConfig.temperature。
func injectTemperature(options map[string]interface{}, protocol string, temperature float64) {
	switch modelOptionPolicyProtocolKey(protocol) {
	case "gemini_generate_content":
		setNestedPath(options, []string{"generationConfig", "temperature"}, temperature)
	default:
		options["temperature"] = temperature
	}
}

// mergeUserModelDefaults 合并用户级模型默认配置到 explicit options，返回新的 options map。
// 优先级（高→低）：explicit > 单模型独立配置 > 用户全局默认 > 平台 defaultOptions（由 filterModelOptions 后续处理）。
func (s *Service) mergeUserModelDefaults(ctx context.Context, userID uint, platformModelName string, explicit map[string]interface{}, protocol string) map[string]interface{} {
	result := cloneModelOptionMap(explicit)
	if result == nil {
		result = make(map[string]interface{})
	}

	platformModelName = strings.TrimSpace(platformModelName)
	if platformModelName == "" {
		return result
	}

	// 读单模型独立配置（chat.model_option:<name>）。
	var singlePayload appusersettings.ModelOptionPayload
	singleKey := appusersettings.ModelOptionKey(platformModelName)
	if raw, err := s.getUserSettingCached(ctx, userID, singleKey); err == nil && strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &singlePayload); err != nil && s.logger != nil {
			s.logger.Warn(fmt.Sprintf("parse user model option payload failed for %s: %v", platformModelName, err))
		}
	}

	// 读用户全局默认三 key。
	defaults := s.getUserSettingsCached(ctx, userID, []string{
		userSettingKeyDefaultThinkingEnabled,
		userSettingKeyDefaultTemperature,
		userSettingKeyDefaultReasoningEffort,
	})

	// 思考开关解析：单模型 > 全局；explicit 显式优先。
	enabledExplicit, hasExplicitEnabled := hasExplicitThinkingEnabled(result)
	var thinkingEnabled *bool
	if hasExplicitEnabled {
		// explicit 已表达，不覆盖。但仍需记录用于后续 strip 决策。
		captured := enabledExplicit
		thinkingEnabled = &captured
	} else if singlePayload.ThinkingEnabled != nil {
		captured := *singlePayload.ThinkingEnabled
		thinkingEnabled = &captured
	} else if raw, ok := defaults[userSettingKeyDefaultThinkingEnabled]; ok {
		captured := raw != "false"
		thinkingEnabled = &captured
	}

	// 注入单模型温度/思考等级（explicit 未设时）。
	if singlePayload.Temperature != nil && !hasExplicitTemperature(result) {
		injectTemperature(result, protocol, *singlePayload.Temperature)
	}
	if singlePayload.ReasoningEffort != nil && !hasExplicitReasoningEffort(result) {
		injectReasoningEffort(result, protocol, *singlePayload.ReasoningEffort)
	}

	// 注入全局默认温度/思考等级（explicit 与单模型都未设时）。
	if !hasExplicitTemperature(result) && singlePayload.Temperature == nil {
		if raw, ok := defaults[userSettingKeyDefaultTemperature]; ok {
			if temp, parseErr := parseFloatRaw(raw); parseErr == nil {
				injectTemperature(result, protocol, temp)
			}
		}
	}
	if !hasExplicitReasoningEffort(result) && singlePayload.ReasoningEffort == nil {
		if raw, ok := defaults[userSettingKeyDefaultReasoningEffort]; ok {
			effort := strings.TrimSpace(raw)
			if effort != "" {
				injectReasoningEffort(result, protocol, effort)
			}
		}
	}

	// 思考开关关闭时清除所有思考字段；开启时按协议注入开关字段（anthropic）。
	if thinkingEnabled != nil {
		if !*thinkingEnabled {
			stripThinkingFields(result)
		} else if !hasExplicitEnabled {
			injectThinkingEnabled(result, protocol, true)
		}
	}
	// enable_thinking 在非 Anthropic 协议中仅作为用户级开关控制信号，不应透传给上游。
	if protocolThinkingEnabledPath(protocol) == "" {
		delete(result, "enable_thinking")
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func parseFloatRaw(raw string) (float64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("empty")
	}
	var f float64
	if _, err := fmt.Sscanf(trimmed, "%f", &f); err != nil {
		return 0, err
	}
	return f, nil
}
