package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iiwish/data-anonymization/internal/config"
)

func init() {
	// 设置测试配置
	gin.SetMode(gin.TestMode)

	// 加载配置（用于测试）
	config.Load("../../config.example.json")
}

func TestHMACAuth_Success(t *testing.T) {
	router := gin.New()
	router.POST("/test", HMACAuth(300), func(c *gin.Context) {
		systemID, _ := c.Get("system_id")
		userID, _ := c.Get("user_id")
		c.JSON(200, gin.H{
			"system_id": systemID,
			"user_id":   userID,
			"message":   "success",
		})
	})

	// 准备请求体
	requestBody := `{"test":"data"}`
	bodyHash := sha256.Sum256([]byte(requestBody))
	bodyHashStr := hex.EncodeToString(bodyHash[:])

	// 生成签名
	systemID := "BI_REPORT_SYSTEM"
	userID := "user123"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	secret := "a_very_strong_and_long_secret_for_bi"

	signContent := systemID + userID + timestamp + bodyHashStr
	signature := calculateHMAC(secret, signContent)

	// 构建Authorization头
	authHeader := fmt.Sprintf("MCP-HMAC-SHA256 SystemID=%s,UserID=%s,Timestamp=%s,Signature=%s",
		systemID, userID, timestamp, signature)

	// 创建请求
	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(requestBody))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	if w.Code != 200 {
		t.Errorf("期望状态码 200，得到 %d, 响应: %s", w.Code, w.Body.String())
	}
}

func TestHMACAuth_MissingAuthHeader(t *testing.T) {
	router := gin.New()
	router.POST("/test", HMACAuth(300), func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("期望状态码 401，得到 %d", w.Code)
	}
}

func TestHMACAuth_InvalidAuthFormat(t *testing.T) {
	router := gin.New()
	router.POST("/test", HMACAuth(300), func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"test":"data"}`))
	req.Header.Set("Authorization", "Invalid Format")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("期望状态码 401，得到 %d", w.Code)
	}
}

func TestHMACAuth_InvalidSystemID(t *testing.T) {
	router := gin.New()
	router.POST("/test", HMACAuth(300), func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	requestBody := `{"test":"data"}`
	bodyHash := sha256.Sum256([]byte(requestBody))
	bodyHashStr := hex.EncodeToString(bodyHash[:])

	systemID := "INVALID_SYSTEM"
	userID := "user123"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	secret := "some_secret"

	signContent := systemID + userID + timestamp + bodyHashStr
	signature := calculateHMAC(secret, signContent)

	authHeader := fmt.Sprintf("MCP-HMAC-SHA256 SystemID=%s,UserID=%s,Timestamp=%s,Signature=%s",
		systemID, userID, timestamp, signature)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(requestBody))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("期望状态码 401，得到 %d", w.Code)
	}
}

func TestHMACAuth_ExpiredTimestamp(t *testing.T) {
	router := gin.New()
	router.POST("/test", HMACAuth(300), func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	requestBody := `{"test":"data"}`
	bodyHash := sha256.Sum256([]byte(requestBody))
	bodyHashStr := hex.EncodeToString(bodyHash[:])

	systemID := "BI_REPORT_SYSTEM"
	userID := "user123"
	// 使用过期的时间戳（1小时前）
	timestamp := strconv.FormatInt(time.Now().Unix()-3600, 10)
	secret := "a_very_strong_and_long_secret_for_bi"

	signContent := systemID + userID + timestamp + bodyHashStr
	signature := calculateHMAC(secret, signContent)

	authHeader := fmt.Sprintf("MCP-HMAC-SHA256 SystemID=%s,UserID=%s,Timestamp=%s,Signature=%s",
		systemID, userID, timestamp, signature)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(requestBody))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("期望状态码 401，得到 %d", w.Code)
	}
}

func TestHMACAuth_InvalidSignature(t *testing.T) {
	router := gin.New()
	router.POST("/test", HMACAuth(300), func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	requestBody := `{"test":"data"}`

	systemID := "BI_REPORT_SYSTEM"
	userID := "user123"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	// 使用错误的签名
	signature := "invalid_signature_here"

	authHeader := fmt.Sprintf("MCP-HMAC-SHA256 SystemID=%s,UserID=%s,Timestamp=%s,Signature=%s",
		systemID, userID, timestamp, signature)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(requestBody))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("期望状态码 401，得到 %d", w.Code)
	}
}

func TestHMACAuth_MissingParameters(t *testing.T) {
	router := gin.New()
	router.POST("/test", HMACAuth(300), func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// 缺少UserID参数
	systemID := "BI_REPORT_SYSTEM"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := "some_signature"

	authHeader := fmt.Sprintf("MCP-HMAC-SHA256 SystemID=%s,Timestamp=%s,Signature=%s",
		systemID, timestamp, signature)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"test":"data"}`))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("期望状态码 401，得到 %d", w.Code)
	}
}

func TestHMACAuth_InvalidTimestampFormat(t *testing.T) {
	router := gin.New()
	router.POST("/test", HMACAuth(300), func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	requestBody := `{"test":"data"}`

	systemID := "BI_REPORT_SYSTEM"
	userID := "user123"
	timestamp := "invalid_timestamp"
	signature := "some_signature"

	authHeader := fmt.Sprintf("MCP-HMAC-SHA256 SystemID=%s,UserID=%s,Timestamp=%s,Signature=%s",
		systemID, userID, timestamp, signature)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(requestBody))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("期望状态码 401，得到 %d", w.Code)
	}
}

func TestParseAuthHeader(t *testing.T) {
	tests := []struct {
		name        string
		header      string
		expectError bool
		expectedLen int
	}{
		{
			name:        "Valid header",
			header:      "MCP-HMAC-SHA256 SystemID=TEST,UserID=user1,Timestamp=123,Signature=abc",
			expectError: false,
			expectedLen: 4,
		},
		{
			name:        "Invalid format - no space",
			header:      "MCP-HMAC-SHA256",
			expectError: true,
			expectedLen: 0,
		},
		{
			name:        "Invalid auth type",
			header:      "Bearer token123",
			expectError: true,
			expectedLen: 0,
		},
		{
			name:        "Empty header",
			header:      "",
			expectError: true,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := parseAuthHeader(tt.header)

			if tt.expectError {
				if err == nil {
					t.Error("期望错误，但没有得到错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
				if len(params) != tt.expectedLen {
					t.Errorf("期望 %d 个参数，得到 %d 个", tt.expectedLen, len(params))
				}
			}
		})
	}
}

func TestCalculateHMAC(t *testing.T) {
	secret := "test_secret"
	message := "test_message"

	signature1 := calculateHMAC(secret, message)
	signature2 := calculateHMAC(secret, message)

	// 相同的输入应该产生相同的签名
	if signature1 != signature2 {
		t.Error("相同的输入应该产生相同的签名")
	}

	// 不同的消息应该产生不同的签名
	signature3 := calculateHMAC(secret, "different_message")
	if signature1 == signature3 {
		t.Error("不同的消息应该产生不同的签名")
	}

	// 不同的密钥应该产生不同的签名
	signature4 := calculateHMAC("different_secret", message)
	if signature1 == signature4 {
		t.Error("不同的密钥应该产生不同的签名")
	}

	// 验证签名格式（应该是64个十六进制字符）
	if len(signature1) != 64 {
		t.Errorf("签名长度应该是64，得到 %d", len(signature1))
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int64
		expected int64
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{100, 100},
		{-100, 100},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.expected {
			t.Errorf("abs(%d) = %d; 期望 %d", tt.input, result, tt.expected)
		}
	}
}

func TestHMACAuth_DifferentSystems(t *testing.T) {
	router := gin.New()
	router.POST("/test", HMACAuth(300), func(c *gin.Context) {
		systemID, _ := c.Get("system_id")
		c.JSON(200, gin.H{"system_id": systemID})
	})

	systems := []struct {
		systemID string
		secret   string
	}{
		{"BI_REPORT_SYSTEM", "a_very_strong_and_long_secret_for_bi"},
		{"CUSTOMER_SERVICE_BOT", "another_unique_secret_for_the_chatbot"},
	}

	for _, sys := range systems {
		requestBody := `{"test":"data"}`
		bodyHash := sha256.Sum256([]byte(requestBody))
		bodyHashStr := hex.EncodeToString(bodyHash[:])

		userID := "user123"
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)

		signContent := sys.systemID + userID + timestamp + bodyHashStr
		signature := calculateHMAC(sys.secret, signContent)

		authHeader := fmt.Sprintf("MCP-HMAC-SHA256 SystemID=%s,UserID=%s,Timestamp=%s,Signature=%s",
			sys.systemID, userID, timestamp, signature)

		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(requestBody))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("系统 %s 认证失败，状态码: %d", sys.systemID, w.Code)
		}
	}
}
