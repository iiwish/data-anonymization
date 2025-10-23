package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iiwish/data-anonymization/internal/config"
	"github.com/iiwish/data-anonymization/internal/logger"
)

// AuthResponse 鉴权失败响应
type AuthResponse struct {
	Error string `json:"error"`
}

// HMACAuth HMAC鉴权中间件
func HMACAuth(timestampWindow int) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// 解析Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.LogRequest("Auth", "", "", "FAILED", 0, "缺少Authorization头")
			c.JSON(http.StatusUnauthorized, AuthResponse{Error: "缺少Authorization头"})
			c.Abort()
			return
		}

		// 解析认证参数
		params, err := parseAuthHeader(authHeader)
		if err != nil {
			logger.LogRequest("Auth", "", "", "FAILED", 0, err.Error())
			c.JSON(http.StatusUnauthorized, AuthResponse{Error: err.Error()})
			c.Abort()
			return
		}

		systemID := params["SystemID"]
		userID := params["UserID"]
		timestampStr := params["Timestamp"]
		signature := params["Signature"]

		// 验证必需参数
		if systemID == "" || userID == "" || timestampStr == "" || signature == "" {
			logger.LogRequest("Auth", systemID, userID, "FAILED", 0, "缺少必需的认证参数")
			c.JSON(http.StatusUnauthorized, AuthResponse{Error: "缺少必需的认证参数"})
			c.Abort()
			return
		}

		// 查找系统配置
		sysConfig, ok := config.GetSystemConfig(systemID)
		if !ok {
			logger.LogRequest("Auth", systemID, userID, "FAILED", 0, "无效的SystemID")
			c.JSON(http.StatusUnauthorized, AuthResponse{Error: "无效的SystemID"})
			c.Abort()
			return
		}

		// 验证时间戳
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			logger.LogRequest("Auth", systemID, userID, "FAILED", 0, "无效的时间戳格式")
			c.JSON(http.StatusUnauthorized, AuthResponse{Error: "无效的时间戳格式"})
			c.Abort()
			return
		}

		now := time.Now().Unix()
		if abs(now-timestamp) > int64(timestampWindow) {
			logger.LogRequest("Auth", systemID, userID, "FAILED", 0, "时间戳已过期")
			c.JSON(http.StatusUnauthorized, AuthResponse{Error: "时间戳已过期"})
			c.Abort()
			return
		}

		// 读取请求体并计算其SHA256
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logger.LogRequest("Auth", systemID, userID, "FAILED", 0, "读取请求体失败")
			c.JSON(http.StatusBadRequest, AuthResponse{Error: "读取请求体失败"})
			c.Abort()
			return
		}

		// 重新设置请求体，以便后续处理器可以读取
		c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

		// 计算请求体的SHA256
		bodyHash := sha256.Sum256(bodyBytes)
		bodyHashStr := hex.EncodeToString(bodyHash[:])

		// 构建签名内容
		signContent := systemID + userID + timestampStr + bodyHashStr

		// 使用系统密钥计算HMAC-SHA256
		expectedSignature := calculateHMAC(sysConfig.SharedSecret, signContent)

		// 比对签名
		if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			latency := time.Since(startTime).Milliseconds()
			logger.LogRequest("Auth", systemID, userID, "FAILED", latency, "签名验证失败")
			c.JSON(http.StatusUnauthorized, AuthResponse{Error: "签名验证失败"})
			c.Abort()
			return
		}

		// 鉴权成功，将系统信息存入上下文
		c.Set("system_id", systemID)
		c.Set("user_id", userID)
		c.Set("auth_start_time", startTime)

		c.Next()
	}
}

// parseAuthHeader 解析Authorization头
// 格式: MCP-HMAC-SHA256 SystemID={...},UserID={...},Timestamp={...},Signature={...}
func parseAuthHeader(authHeader string) (map[string]string, error) {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("无效的Authorization格式")
	}

	if parts[0] != "MCP-HMAC-SHA256" {
		return nil, fmt.Errorf("不支持的认证方式: %s", parts[0])
	}

	params := make(map[string]string)
	pairs := strings.Split(parts[1], ",")

	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}

	return params, nil
}

// calculateHMAC 计算HMAC-SHA256签名
func calculateHMAC(secret, message string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// abs 返回整数的绝对值
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
