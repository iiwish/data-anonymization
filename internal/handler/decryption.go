package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iiwish/data-anonymization/internal/logger"
	"github.com/iiwish/data-anonymization/internal/service"
)

// DecryptionHandler 解密服务处理器
type DecryptionHandler struct{}

// NewDecryptionHandler 创建新的解密处理器
func NewDecryptionHandler() *DecryptionHandler {
	return &DecryptionHandler{}
}

// Handle 处理解密请求
func (h *DecryptionHandler) Handle(c *gin.Context) {
	startTime, _ := c.Get("auth_start_time")
	start := startTime.(time.Time)

	systemID, _ := c.Get("system_id")
	userID, _ := c.Get("user_id")

	var req service.DecryptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		latency := time.Since(start).Milliseconds()
		logger.LogRequest("DecryptionService", systemID.(string), userID.(string), "FAILED", latency, "无效的请求格式: "+err.Error())
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的请求格式: " + err.Error()})
		return
	}

	// 验证必需字段
	if req.DataWithAnonymizedCodes == nil {
		latency := time.Since(start).Milliseconds()
		logger.LogRequest("DecryptionService", systemID.(string), userID.(string), "FAILED", latency, "data_with_anonymized_codes不能为空")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "data_with_anonymized_codes不能为空"})
		return
	}

	if req.Mappings == nil || len(req.Mappings) == 0 {
		latency := time.Since(start).Milliseconds()
		logger.LogRequest("DecryptionService", systemID.(string), userID.(string), "FAILED", latency, "mappings不能为空")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "mappings不能为空"})
		return
	}

	// 创建解密器
	decryptor := service.NewDecryptor(req.Mappings)

	// 执行解密
	decryptedData := decryptor.Decrypt(req.DataWithAnonymizedCodes)

	// 构造响应
	response := service.DecryptionResponse{
		DecryptedData: decryptedData,
	}

	latency := time.Since(start).Milliseconds()
	logger.LogRequest("DecryptionService", systemID.(string), userID.(string), "SUCCESS", latency, "")

	c.JSON(http.StatusOK, response)
}
