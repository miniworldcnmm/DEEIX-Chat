package usersettings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	domainusersettings "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/usersettings"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

// ErrValidation 表示用户输入校验失败，可被 handler 识别并返回 400。
type ErrValidation struct {
	Msg string
}

func (e *ErrValidation) Error() string { return e.Msg }

// modelOptionKeyPrefix 是单模型独立默认配置在 user_settings 表中的 key 前缀。
// 完整 key 形如 "chat.model_option:<platformModelName>"，value 为 JSON 字符串。
const modelOptionKeyPrefix = "chat.model_option:"

// ModelOptionPayload 是单模型独立默认配置的可视化结构。
type ModelOptionPayload struct {
	ThinkingEnabled *bool    `json:"thinking_enabled,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
	ReasoningEffort *string  `json:"reasoning_effort,omitempty"`
}

// allowedKeys 是用户可配置的 key 集合及其默认值。
var allowedKeys = map[string]string{
	"chat.file_mode":                            "auto",
	"chat.send_on_enter":                        "enter",
	"chat.show_token_usage":                     "true",
	"chat.show_model_info":                      "true",
	"chat.show_latency":                         "true",
	"chat.show_billing_cost":                    "true",
	"chat.default_model":                        "",
	"chat.auto_generate_title":                  "true",
	"chat.delete_conversation_files_by_default": "false",
	"chat.context_compact_auto":                 "true",
	"chat.markdown_render":                      "true",
	"chat.restore_draft_on_failure":             "true",
	"chat.preserve_conversation_drafts":         "true",
	"chat.reuse_model_options":                  "true",
	"chat.input_height":                         "standard",
	"chat.content_width":                        "compact",
	"chat.default_mcp_tool_ids":                 "[]",
	"chat.default_thinking_enabled":             "true",
	"chat.default_temperature":                  "1",
	"chat.default_reasoning_effort":             "",
	"chat.memory_enabled":                       "false",
}

// boolKeys 取值只能是 "true" / "false"。
var boolKeys = map[string]bool{
	"chat.show_token_usage":                     true,
	"chat.show_model_info":                      true,
	"chat.show_latency":                         true,
	"chat.show_billing_cost":                    true,
	"chat.auto_generate_title":                  true,
	"chat.delete_conversation_files_by_default": true,
	"chat.context_compact_auto":                 true,
	"chat.markdown_render":                      true,
	"chat.restore_draft_on_failure":             true,
	"chat.preserve_conversation_drafts":         true,
	"chat.reuse_model_options":                  true,
	"chat.default_thinking_enabled":             true,
	"chat.memory_enabled":                       true,
}

// enumKeys 枚举 key 的合法值集合。
var enumKeys = map[string]map[string]bool{
	"chat.file_mode":     {"auto": true, "full_context": true, "rag": true},
	"chat.send_on_enter": {"enter": true, "ctrl_enter": true, "meta_enter": true},
	"chat.input_height":  {"compact": true, "standard": true, "loose": true},
	"chat.content_width": {"compact": true, "standard": true, "wide": true},
}

const (
	defaultTemperatureMin = 0.0
	defaultTemperatureMax = 2.0
	platformModelNameMax  = 128
)

// validateValue 校验 key 对应 value 的合法性。
func validateValue(key, value string) error {
	if key == "chat.default_mcp_tool_ids" {
		return validateDefaultMCPToolIDs(value, key)
	}
	if key == "chat.default_temperature" {
		return validateDefaultTemperature(value, key)
	}
	if strings.HasPrefix(key, modelOptionKeyPrefix) {
		return validateModelOptionPayload(key, value)
	}
	if boolKeys[key] {
		if value != "true" && value != "false" {
			return &ErrValidation{Msg: fmt.Sprintf("invalid value for %s: must be 'true' or 'false'", key)}
		}
	}
	if allowed, ok := enumKeys[key]; ok {
		if !allowed[value] {
			valid := make([]string, 0, len(allowed))
			for v := range allowed {
				valid = append(valid, "'"+v+"'")
			}
			return &ErrValidation{Msg: fmt.Sprintf("invalid value for %s: must be one of %s", key, strings.Join(valid, ", "))}
		}
	}
	return nil
}

func validateDefaultTemperature(value string, key string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return &ErrValidation{Msg: fmt.Sprintf("invalid value for %s: must be a number in [0, 2]", key)}
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return &ErrValidation{Msg: fmt.Sprintf("invalid value for %s: must be a number in [0, 2]", key)}
	}
	if parsed < defaultTemperatureMin || parsed > defaultTemperatureMax {
		return &ErrValidation{Msg: fmt.Sprintf("invalid value for %s: must be in [0, 2]", key)}
	}
	return nil
}

// validateModelOptionPayload 校验单模型独立默认配置的 value(JSON)。
func validateModelOptionPayload(key, value string) error {
	platformModelName := strings.TrimPrefix(key, modelOptionKeyPrefix)
	if strings.TrimSpace(platformModelName) == "" || len(platformModelName) > platformModelNameMax {
		return &ErrValidation{Msg: fmt.Sprintf("invalid model option key: %s", key)}
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return &ErrValidation{Msg: "model option payload must not be empty"}
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return &ErrValidation{Msg: fmt.Sprintf("invalid model option payload: %v", err)}
	}
	for k := range raw {
		switch k {
		case "thinking_enabled", "temperature", "reasoning_effort":
		default:
			return &ErrValidation{Msg: fmt.Sprintf("unexpected model option field: %s", k)}
		}
	}
	if v, ok := raw["thinking_enabled"]; ok {
		if _, ok := v.(bool); !ok {
			return &ErrValidation{Msg: "model option field thinking_enabled must be bool"}
		}
	}
	if v, ok := raw["temperature"]; ok {
		f, ok := v.(float64)
		if !ok {
			return &ErrValidation{Msg: "model option field temperature must be number"}
		}
		if f < defaultTemperatureMin || f > defaultTemperatureMax {
			return &ErrValidation{Msg: "model option field temperature must be in [0, 2]"}
		}
	}
	if v, ok := raw["reasoning_effort"]; ok {
		if _, ok := v.(string); !ok {
			return &ErrValidation{Msg: "model option field reasoning_effort must be string"}
		}
	}
	return nil
}

func validateDefaultMCPToolIDs(value string, key string) error {
	var toolIDs []uint64
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &toolIDs); err != nil {
		return &ErrValidation{Msg: fmt.Sprintf("invalid value for %s: must be a JSON array of positive tool IDs", key)}
	}
	if len(toolIDs) > 128 {
		return &ErrValidation{Msg: fmt.Sprintf("invalid value for %s: must contain at most 128 tool IDs", key)}
	}
	for _, id := range toolIDs {
		if id == 0 {
			return &ErrValidation{Msg: fmt.Sprintf("invalid value for %s: tool IDs must be positive integers", key)}
		}
	}
	return nil
}

// IsValidationError 判断 err 是否为校验错误。
func IsValidationError(err error) bool {
	var ve *ErrValidation
	return errors.As(err, &ve)
}

// UserSettingCacheInvalidator 由 conversation.Service 注入，用于在用户设置变更后失效其内部缓存。
type UserSettingCacheInvalidator func(userID uint, key string)

// Service 封装用户配置业务逻辑。
type Service struct {
	repo            repository.UserSettingsRepository
	invalidateCache UserSettingCacheInvalidator
}

// NewService 创建服务。
func NewService(repo repository.UserSettingsRepository) *Service {
	return &Service{repo: repo}
}

// SetUserSettingCacheInvalidator 注入缓存失效回调。
func (s *Service) SetUserSettingCacheInvalidator(fn UserSettingCacheInvalidator) {
	s.invalidateCache = fn
}

func (s *Service) notifyInvalidate(userID uint, key string) {
	if s.invalidateCache != nil {
		s.invalidateCache(userID, key)
	}
}

// ListSettings 返回指定用户的全部配置，缺失的 key 用默认值填充。
func (s *Service) ListSettings(ctx context.Context, userID uint) (map[string]string, error) {
	rows, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(allowedKeys))
	// 填充默认值
	for k, v := range allowedKeys {
		result[k] = v
	}
	// 覆盖用户设置值
	for _, row := range rows {
		if _, ok := allowedKeys[row.Key]; ok {
			result[row.Key] = row.Value
		}
	}
	return result, nil
}

// PatchSettings 批量更新用户配置项，返回更新后的全量配置。
func (s *Service) PatchSettings(ctx context.Context, userID uint, patches map[string]string) (map[string]string, error) {
	now := time.Now()
	items := make([]domainusersettings.UserSetting, 0, len(patches))
	for key, value := range patches {
		key = strings.TrimSpace(key)
		if _, ok := allowedKeys[key]; !ok {
			return nil, &ErrValidation{Msg: fmt.Sprintf("unknown setting key: %s", key)}
		}
		if err := validateValue(key, value); err != nil {
			return nil, err
		}
		items = append(items, domainusersettings.UserSetting{
			UserID:    userID,
			Key:       key,
			Value:     value,
			UpdatedAt: now,
		})
	}
	if err := s.repo.Upsert(ctx, items); err != nil {
		return nil, err
	}
	for _, item := range items {
		s.notifyInvalidate(userID, item.Key)
	}
	return s.ListSettings(ctx, userID)
}

// ModelOptionKey 拼接单模型独立默认配置的存储 key。
func ModelOptionKey(platformModelName string) string {
	return modelOptionKeyPrefix + platformModelName
}

// ListUserModelOptions 列出当前用户所有单模型独立默认配置。
func (s *Service) ListUserModelOptions(ctx context.Context, userID uint) (map[string]ModelOptionPayload, error) {
	rows, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]ModelOptionPayload)
	for _, row := range rows {
		if !strings.HasPrefix(row.Key, modelOptionKeyPrefix) {
			continue
		}
		name := strings.TrimPrefix(row.Key, modelOptionKeyPrefix)
		if strings.TrimSpace(name) == "" {
			continue
		}
		trimmed := strings.TrimSpace(row.Value)
		if trimmed == "" {
			result[name] = ModelOptionPayload{}
			continue
		}
		var payload ModelOptionPayload
		if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
			continue
		}
		result[name] = payload
	}
	return result, nil
}

// UpsertUserModelOption 写入或覆盖一个单模型独立默认配置。
func (s *Service) UpsertUserModelOption(ctx context.Context, userID uint, platformModelName string, payload ModelOptionPayload) error {
	name := strings.TrimSpace(platformModelName)
	if name == "" || len(name) > platformModelNameMax {
		return &ErrValidation{Msg: "invalid platform model name"}
	}
	key := ModelOptionKey(name)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	value := string(encoded)
	if err := validateModelOptionPayload(key, value); err != nil {
		return err
	}
	if err := s.repo.Upsert(ctx, []domainusersettings.UserSetting{
		{UserID: userID, Key: key, Value: value, UpdatedAt: time.Now()},
	}); err != nil {
		return err
	}
	s.notifyInvalidate(userID, key)
	return nil
}

// DeleteUserModelOption 清除一个单模型独立默认配置。
func (s *Service) DeleteUserModelOption(ctx context.Context, userID uint, platformModelName string) error {
	name := strings.TrimSpace(platformModelName)
	if name == "" {
		return &ErrValidation{Msg: "invalid platform model name"}
	}
	key := ModelOptionKey(name)
	if err := s.repo.Delete(ctx, userID, key); err != nil {
		return err
	}
	s.notifyInvalidate(userID, key)
	return nil
}
