package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iiwish/data-anonymization/internal/logger"
	"github.com/iiwish/data-anonymization/internal/service"
)

// AnonymizationHandler 匿名化服务处理器
type AnonymizationHandler struct{}

// NewAnonymizationHandler 创建新的匿名化处理器
func NewAnonymizationHandler() *AnonymizationHandler {
	return &AnonymizationHandler{}
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}

// Handle 处理匿名化请求
func (h *AnonymizationHandler) Handle(c *gin.Context) {
	startTime, _ := c.Get("auth_start_time")
	start := startTime.(time.Time)

	systemID, _ := c.Get("system_id")
	userID, _ := c.Get("user_id")

	var req service.AnonymizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		latency := time.Since(start).Milliseconds()
		logger.LogRequest("AnonymizationService", systemID.(string), userID.(string), "FAILED", latency, "无效的请求格式: "+err.Error())
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的请求格式: " + err.Error()})
		return
	}

	// 验证必需字段
	if req.SessionID == "" {
		latency := time.Since(start).Milliseconds()
		logger.LogRequest("AnonymizationService", systemID.(string), userID.(string), "FAILED", latency, "session_id不能为空")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "session_id不能为空"})
		return
	}

	if req.Payload == nil {
		latency := time.Since(start).Milliseconds()
		logger.LogRequest("AnonymizationService", systemID.(string), userID.(string), "FAILED", latency, "payload不能为空")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "payload不能为空"})
		return
	}

	if len(req.AnonymizationRules) == 0 {
		latency := time.Since(start).Milliseconds()
		logger.LogRequest("AnonymizationService", systemID.(string), userID.(string), "FAILED", latency, "anonymization_rules不能为空")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "anonymization_rules不能为空"})
		return
	}

	// 创建匿名化器
	anonymizer := service.NewAnonymizer(req.AnonymizationRules)

	// 执行匿名化
	anonymizedPayload, err := anonymizer.Anonymize(req.Payload)
	if err != nil {
		latency := time.Since(start).Milliseconds()
		logger.LogRequest("AnonymizationService", systemID.(string), userID.(string), "FAILED", latency, "匿名化失败: "+err.Error())
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "匿名化失败: " + err.Error()})
		return
	}

	// 获取映射表
	mappings := anonymizer.GetMappings()

	// 构造响应
	response := service.AnonymizationResponse{
		SessionID:         req.SessionID,
		AnonymizedPayload: anonymizedPayload,
		MappingsToStore:   mappings,
	}

	latency := time.Since(start).Milliseconds()
	logger.LogRequest("AnonymizationService", systemID.(string), userID.(string), "SUCCESS", latency, "")

	c.JSON(http.StatusOK, response)
}
