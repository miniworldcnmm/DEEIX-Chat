package usersettings

import (
	"net/http"
	"strings"

	appusersettings "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/usersettings"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// Handler 封装用户配置 HTTP 处理。
type Handler struct {
	service *appusersettings.Service
}

// NewHandler 创建处理器。
func NewHandler(service *appusersettings.Service) *Handler {
	return &Handler{service: service}
}

// GetSettings godoc
// @Summary 获取当前用户的配置
// @Description 返回当前用户全部个人偏好配置，缺失项以默认值填充
// @Tags user/settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserSettingsResponseDoc
// @Failure 500 {object} response.Envelope
// @Router /user/settings [get]
func (h *Handler) GetSettings(c *gin.Context) {
	userID := middleware.MustUserID(c)
	data, err := h.service.ListSettings(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "load settings failed")
		return
	}
	response.Success(c, UserSettingsResponse{Settings: data})
}

// PatchSettings godoc
// @Summary 更新当前用户的配置
// @Description 批量更新用户个人偏好配置，返回更新后的全量配置
// @Tags user/settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body PatchSettingsRequest true "更新项"
// @Success 200 {object} UserSettingsResponseDoc
// @Failure 400 {object} response.Envelope
// @Failure 500 {object} response.Envelope
// @Router /user/settings [patch]
func (h *Handler) PatchSettings(c *gin.Context) {
	var req PatchSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}

	userID := middleware.MustUserID(c)
	data, err := h.service.PatchSettings(c.Request.Context(), userID, req.Settings)
	if err != nil {
		if appusersettings.IsValidationError(err) {
			response.ErrorFrom(c, http.StatusBadRequest, err)
		} else {
			response.Error(c, http.StatusInternalServerError, "settings update failed")
		}
		return
	}
	response.Success(c, UserSettingsResponse{Settings: data})
}

// ListUserModelOptions godoc
// @Summary 列出当前用户的单模型独立默认配置
// @Description 返回该用户为每个平台模型单独保存的默认思考开关、温度、思考等级配置
// @Tags user/settings/model-options
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ListUserModelOptionsResponseDoc
// @Failure 500 {object} response.Envelope
// @Router /user/settings/model-options [get]
func (h *Handler) ListUserModelOptions(c *gin.Context) {
	userID := middleware.MustUserID(c)
	options, err := h.service.ListUserModelOptions(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "load model options failed")
		return
	}
	response.Success(c, ListUserModelOptionsResponse{Options: toModelOptionResponses(options)})
}

// UpsertUserModelOption godoc
// @Summary 写入或覆盖单个平台模型的独立默认配置
// @Description 为指定平台模型保存独立的思考开关、温度、思考等级默认配置，不会被后续全局默认覆盖
// @Tags user/settings/model-options
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param platformModelName path string true "平台模型名"
// @Param body body ModelOptionPayloadRequest true "单模型默认配置"
// @Success 200 {object} ListUserModelOptionsResponseDoc
// @Failure 400 {object} response.Envelope
// @Failure 500 {object} response.Envelope
// @Router /user/settings/model-options/{platformModelName} [put]
func (h *Handler) UpsertUserModelOption(c *gin.Context) {
	platformModelName := strings.TrimSpace(c.Param("platformModelName"))
	if platformModelName == "" {
		response.Error(c, http.StatusBadRequest, "invalid platform model name")
		return
	}
	var req ModelOptionPayloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	userID := middleware.MustUserID(c)
	payload := appusersettings.ModelOptionPayload{
		ThinkingEnabled: req.ThinkingEnabled,
		Temperature:     req.Temperature,
		ReasoningEffort: req.ReasoningEffort,
	}
	if err := h.service.UpsertUserModelOption(c.Request.Context(), userID, platformModelName, payload); err != nil {
		if appusersettings.IsValidationError(err) {
			response.ErrorFrom(c, http.StatusBadRequest, err)
		} else {
			response.Error(c, http.StatusInternalServerError, "model option update failed")
		}
		return
	}
	options, err := h.service.ListUserModelOptions(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "load model options failed")
		return
	}
	response.Success(c, ListUserModelOptionsResponse{Options: toModelOptionResponses(options)})
}

// DeleteUserModelOption godoc
// @Summary 清除单个平台模型的独立默认配置
// @Description 删除指定平台模型的独立默认配置，删除后该模型回退到用户全局默认
// @Tags user/settings/model-options
// @Produce json
// @Security BearerAuth
// @Param platformModelName path string true "平台模型名"
// @Success 200 {object} ListUserModelOptionsResponseDoc
// @Failure 400 {object} response.Envelope
// @Failure 500 {object} response.Envelope
// @Router /user/settings/model-options/{platformModelName} [delete]
func (h *Handler) DeleteUserModelOption(c *gin.Context) {
	platformModelName := strings.TrimSpace(c.Param("platformModelName"))
	if platformModelName == "" {
		response.Error(c, http.StatusBadRequest, "invalid platform model name")
		return
	}
	userID := middleware.MustUserID(c)
	if err := h.service.DeleteUserModelOption(c.Request.Context(), userID, platformModelName); err != nil {
		if appusersettings.IsValidationError(err) {
			response.ErrorFrom(c, http.StatusBadRequest, err)
		} else {
			response.Error(c, http.StatusInternalServerError, "model option delete failed")
		}
		return
	}
	options, err := h.service.ListUserModelOptions(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "load model options failed")
		return
	}
	response.Success(c, ListUserModelOptionsResponse{Options: toModelOptionResponses(options)})
}

func toModelOptionResponses(options map[string]appusersettings.ModelOptionPayload) map[string]ModelOptionPayloadResponse {
	result := make(map[string]ModelOptionPayloadResponse, len(options))
	for name, payload := range options {
		result[name] = ModelOptionPayloadResponse{
			ThinkingEnabled: payload.ThinkingEnabled,
			Temperature:     payload.Temperature,
			ReasoningEffort: payload.ReasoningEffort,
		}
	}
	return result
}
