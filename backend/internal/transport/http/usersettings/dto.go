package usersettings

// PatchSettingsRequest 批量更新用户配置请求。
type PatchSettingsRequest struct {
	Settings map[string]string `json:"settings" binding:"required"`
}

// UserSettingsResponse 用户配置响应，键值对形式的全量配置。
type UserSettingsResponse struct {
	Settings map[string]string `json:"settings"`
}

// UserSettingsResponseDoc Swagger 响应文档。
type UserSettingsResponseDoc struct {
	ErrorMsg string               `json:"errorMsg"`
	Data     UserSettingsResponse `json:"data"`
}

// ModelOptionPayloadRequest 单模型独立默认配置写入请求。
type ModelOptionPayloadRequest struct {
	ThinkingEnabled *bool    `json:"thinkingEnabled,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
	ReasoningEffort *string  `json:"reasoningEffort,omitempty"`
}

// ModelOptionPayloadResponse 单模型独立默认配置响应。
type ModelOptionPayloadResponse struct {
	ThinkingEnabled *bool    `json:"thinkingEnabled,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
	ReasoningEffort *string  `json:"reasoningEffort,omitempty"`
}

// ListUserModelOptionsResponse 单模型独立默认配置列表响应。
type ListUserModelOptionsResponse struct {
	Options map[string]ModelOptionPayloadResponse `json:"options"`
}

// ListUserModelOptionsResponseDoc Swagger 响应文档。
type ListUserModelOptionsResponseDoc struct {
	ErrorMsg string                       `json:"errorMsg"`
	Data     ListUserModelOptionsResponse `json:"data"`
}
