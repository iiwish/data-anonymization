package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iiwish/data-anonymization/internal/config"
	"github.com/iiwish/data-anonymization/internal/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
	// 加载测试配置
	config.Load("../../config.example.json")
}

// setupRouter 创建测试路由
func setupRouter() *gin.Engine {
	router := gin.New()

	// 创建处理器
	anonymizationHandler := NewAnonymizationHandler()
	decryptionHandler := NewDecryptionHandler()

	// 配置路由
	v1 := router.Group("/v1")
	{
		// 匿名化接口
		v1.POST("/anonymize", middleware.HMACAuth(300), anonymizationHandler.Handle)

		// 解密接口
		v1.POST("/decrypt", middleware.HMACAuth(300), decryptionHandler.Handle)
	}

	return router
}

// generateSignature 生成HMAC签名
func generateSignature(systemID, userID, secret, requestBody string) string {
	bodyHash := sha256.Sum256([]byte(requestBody))
	bodyHashStr := hex.EncodeToString(bodyHash[:])

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	signContent := systemID + userID + timestamp + bodyHashStr

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signContent))
	signature := hex.EncodeToString(h.Sum(nil))

	return signature
}

// TestAnonymizationEndToEnd 匿名化接口端到端测试
func TestAnonymizationEndToEnd(t *testing.T) {
	router := setupRouter()

	// 准备请求数据
	requestBody := map[string]interface{}{
		"session_id": "test_session_123",
		"payload": map[string]interface{}{
			"metadata": map[string]interface{}{
				"report_name": "Q3 Sales Analysis for 华东",
				"requester":   "user123",
			},
			"data_table": []interface{}{
				map[string]interface{}{
					"区域":  "华东",
					"产品":  "手机",
					"收入":  1500000.0,
					"增长率": "12.5%",
				},
			},
		},
		"anonymization_rules": []interface{}{
			map[string]interface{}{
				"strategy": "MAP_CODE",
				"applies_to": map[string]interface{}{
					"type":   "REGION",
					"values": []interface{}{"华东"},
				},
			},
			map[string]interface{}{
				"strategy": "MAP_CODE",
				"applies_to": map[string]interface{}{
					"type":   "PRODUCT",
					"values": []interface{}{"手机"},
				},
			},
			map[string]interface{}{
				"strategy": "TRANSFORM",
				"strategy_params": map[string]interface{}{
					"noise_level": 0.05,
				},
				"applies_to": map[string]interface{}{
					"type":   "REVENUE",
					"values": []interface{}{1500000.0},
				},
			},
			map[string]interface{}{
				"strategy": "PASSTHROUGH",
				"applies_to": map[string]interface{}{
					"type":   "GROWTH_RATE",
					"values": []interface{}{"12.5%"},
				},
			},
		},
	}

	requestBodyBytes, _ := json.Marshal(requestBody)
	requestBodyStr := string(requestBodyBytes)

	// 生成签名
	systemID := "BI_REPORT_SYSTEM"
	userID := "user123"
	secret := "a_very_strong_and_long_secret_for_bi"
	signature := generateSignature(systemID, userID, secret, requestBodyStr)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	authHeader := "MCP-HMAC-SHA256 SystemID=" + systemID + ",UserID=" + userID + ",Timestamp=" + timestamp + ",Signature=" + signature

	// 创建请求
	req, _ := http.NewRequest("POST", "/v1/anonymize", bytes.NewBufferString(requestBodyStr))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200，得到 %d, 响应: %s", w.Code, w.Body.String())
		return
	}

	// 解析响应
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应结构
	if response["session_id"] != "test_session_123" {
		t.Errorf("session_id不正确，期望 'test_session_123'，得到 %v", response["session_id"])
	}

	// 验证anonymized_payload存在
	anonymizedPayload, ok := response["anonymized_payload"].(map[string]interface{})
	if !ok {
		t.Fatal("anonymized_payload格式不正确")
	}

	// 验证metadata中的report_name已被匿名化
	metadata, ok := anonymizedPayload["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata格式不正确")
	}

	reportName, ok := metadata["report_name"].(string)
	if !ok {
		t.Fatal("report_name应该是字符串")
	}

	if reportName == "Q3 Sales Analysis for 华东" {
		t.Error("report_name应该被匿名化")
	}

	// 验证data_table
	dataTable, ok := anonymizedPayload["data_table"].([]interface{})
	if !ok {
		t.Fatal("data_table格式不正确")
	}

	if len(dataTable) != 1 {
		t.Errorf("data_table应该有1个元素，得到 %d", len(dataTable))
	}

	// 验证mappings_to_store存在
	mappings, ok := response["mappings_to_store"].(map[string]interface{})
	if !ok {
		t.Fatal("mappings_to_store格式不正确")
	}

	// 验证分类映射
	catMappings, ok := mappings["categorical_mappings"].(map[string]interface{})
	if !ok {
		t.Fatal("categorical_mappings格式不正确")
	}

	if _, ok := catMappings["REGION"]; !ok {
		t.Error("缺少REGION映射")
	}

	if _, ok := catMappings["PRODUCT"]; !ok {
		t.Error("缺少PRODUCT映射")
	}
}

// TestDecryptionEndToEnd 解密接口端到端测试
func TestDecryptionEndToEnd(t *testing.T) {
	router := setupRouter()

	// 准备请求数据（使用JSON对象格式）
	requestBody := map[string]interface{}{
		"data_with_anonymized_codes": map[string]interface{}{
			"summary": "REGION_a3f5 区域表现突出，主要贡献来自 PRODUCT_c8b1。",
			"key_findings": []interface{}{
				map[string]interface{}{
					"dimension": "区域",
					"value":     "REGION_a3f5",
				},
				map[string]interface{}{
					"dimension": "产品",
					"value":     "PRODUCT_c8b1",
				},
			},
		},
		"mappings": map[string]interface{}{
			"categorical_mappings": map[string]interface{}{
				"REGION": map[string]interface{}{
					"REGION_a3f5": "华东",
				},
				"PRODUCT": map[string]interface{}{
					"PRODUCT_c8b1": "手机",
				},
			},
		},
	}

	requestBodyBytes, _ := json.Marshal(requestBody)
	requestBodyStr := string(requestBodyBytes)

	// 生成签名
	systemID := "BI_REPORT_SYSTEM"
	userID := "user123"
	secret := "a_very_strong_and_long_secret_for_bi"
	signature := generateSignature(systemID, userID, secret, requestBodyStr)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	authHeader := "MCP-HMAC-SHA256 SystemID=" + systemID + ",UserID=" + userID + ",Timestamp=" + timestamp + ",Signature=" + signature

	// 创建请求
	req, _ := http.NewRequest("POST", "/v1/decrypt", bytes.NewBufferString(requestBodyStr))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200，得到 %d, 响应: %s", w.Code, w.Body.String())
		return
	}

	// 解析响应
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证decrypted_data存在
	decryptedData, ok := response["decrypted_data"].(map[string]interface{})
	if !ok {
		t.Fatal("decrypted_data格式不正确")
	}

	// 验证summary已被解密
	summary, ok := decryptedData["summary"].(string)
	if !ok {
		t.Fatal("summary应该是字符串")
	}

	if !contains(summary, "华东") {
		t.Errorf("summary应该包含 '华东'，但实际是: %s", summary)
	}

	if !contains(summary, "手机") {
		t.Errorf("summary应该包含 '手机'，但实际是: %s", summary)
	}

	// 验证key_findings
	findings, ok := decryptedData["key_findings"].([]interface{})
	if !ok {
		t.Fatal("key_findings应该是数组")
	}

	if len(findings) != 2 {
		t.Fatalf("key_findings应该有2个元素，但有 %d 个", len(findings))
	}

	// 验证第一个finding
	firstFinding, ok := findings[0].(map[string]interface{})
	if !ok {
		t.Fatal("第一个finding应该是map")
	}

	value, ok := firstFinding["value"].(string)
	if !ok {
		t.Fatal("value应该是字符串")
	}

	if value != "华东" {
		t.Errorf("value应该是 '华东'，但是 '%s'", value)
	}
}

// TestDecryptionEndToEnd_TextString 解密纯文本字符串的端到端测试
func TestDecryptionEndToEnd_TextString(t *testing.T) {
	router := setupRouter()

	// 准备请求数据（使用纯文本字符串格式）
	requestBody := map[string]interface{}{
		"data_with_anonymized_codes": "分析显示，REGION_a3f5 区域的 PRODUCT_c8b1 表现最佳。",
		"mappings": map[string]interface{}{
			"categorical_mappings": map[string]interface{}{
				"REGION": map[string]interface{}{
					"REGION_a3f5": "华东",
				},
				"PRODUCT": map[string]interface{}{
					"PRODUCT_c8b1": "手机",
				},
			},
		},
	}

	requestBodyBytes, _ := json.Marshal(requestBody)
	requestBodyStr := string(requestBodyBytes)

	// 生成签名
	systemID := "BI_REPORT_SYSTEM"
	userID := "user123"
	secret := "a_very_strong_and_long_secret_for_bi"
	signature := generateSignature(systemID, userID, secret, requestBodyStr)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	authHeader := "MCP-HMAC-SHA256 SystemID=" + systemID + ",UserID=" + userID + ",Timestamp=" + timestamp + ",Signature=" + signature

	// 创建请求
	req, _ := http.NewRequest("POST", "/v1/decrypt", bytes.NewBufferString(requestBodyStr))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200，得到 %d, 响应: %s", w.Code, w.Body.String())
		return
	}

	// 解析响应
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证decrypted_data是字符串
	decryptedData, ok := response["decrypted_data"].(string)
	if !ok {
		t.Fatal("decrypted_data应该是字符串")
	}

	// 验证已被解密
	if !contains(decryptedData, "华东") {
		t.Errorf("解密后的文本应该包含 '华东'，但实际是: %s", decryptedData)
	}

	if !contains(decryptedData, "手机") {
		t.Errorf("解密后的文本应该包含 '手机'，但实际是: %s", decryptedData)
	}

	// 验证编码已被替换
	if contains(decryptedData, "REGION_a3f5") {
		t.Errorf("解密后的文本不应该包含 'REGION_a3f5'，但实际是: %s", decryptedData)
	}

	if contains(decryptedData, "PRODUCT_c8b1") {
		t.Errorf("解密后的文本不应该包含 'PRODUCT_c8b1'，但实际是: %s", decryptedData)
	}
}

// TestAnonymizationAndDecryptionIntegration 匿名化和解密的完整集成测试
func TestAnonymizationAndDecryptionIntegration(t *testing.T) {
	router := setupRouter()

	// 第一步：匿名化
	originalPayload := map[string]interface{}{
		"metadata": map[string]interface{}{
			"report_name": "Q3 Sales Analysis for 华东",
			"requester":   "user123",
		},
		"data_table": []interface{}{
			map[string]interface{}{
				"区域":  "华东",
				"产品":  "手机",
				"收入":  1500000.0,
				"增长率": "12.5%",
			},
		},
	}

	anonymizeRequest := map[string]interface{}{
		"session_id": "integration_test_session",
		"payload":    originalPayload,
		"anonymization_rules": []interface{}{
			map[string]interface{}{
				"strategy": "MAP_CODE",
				"applies_to": map[string]interface{}{
					"type":   "REGION",
					"values": []interface{}{"华东"},
				},
			},
			map[string]interface{}{
				"strategy": "MAP_CODE",
				"applies_to": map[string]interface{}{
					"type":   "PRODUCT",
					"values": []interface{}{"手机"},
				},
			},
		},
	}

	anonymizeBodyBytes, _ := json.Marshal(anonymizeRequest)
	anonymizeBodyStr := string(anonymizeBodyBytes)

	systemID := "BI_REPORT_SYSTEM"
	userID := "user123"
	secret := "a_very_strong_and_long_secret_for_bi"
	signature := generateSignature(systemID, userID, secret, anonymizeBodyStr)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	authHeader := "MCP-HMAC-SHA256 SystemID=" + systemID + ",UserID=" + userID + ",Timestamp=" + timestamp + ",Signature=" + signature

	// 执行匿名化请求
	req, _ := http.NewRequest("POST", "/v1/anonymize", bytes.NewBufferString(anonymizeBodyStr))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("匿名化请求失败，状态码: %d, 响应: %s", w.Code, w.Body.String())
	}

	// 解析匿名化响应
	var anonymizeResponse map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &anonymizeResponse); err != nil {
		t.Fatalf("解析匿名化响应失败: %v", err)
	}

	anonymizedPayload := anonymizeResponse["anonymized_payload"].(map[string]interface{})
	mappings := anonymizeResponse["mappings_to_store"].(map[string]interface{})

	// 第二步：解密
	decryptRequest := map[string]interface{}{
		"data_with_anonymized_codes": anonymizedPayload,
		"mappings":                   mappings,
	}

	decryptBodyBytes, _ := json.Marshal(decryptRequest)
	decryptBodyStr := string(decryptBodyBytes)

	signature2 := generateSignature(systemID, userID, secret, decryptBodyStr)
	timestamp2 := strconv.FormatInt(time.Now().Unix(), 10)

	authHeader2 := "MCP-HMAC-SHA256 SystemID=" + systemID + ",UserID=" + userID + ",Timestamp=" + timestamp2 + ",Signature=" + signature2

	// 执行解密请求
	req2, _ := http.NewRequest("POST", "/v1/decrypt", bytes.NewBufferString(decryptBodyStr))
	req2.Header.Set("Authorization", authHeader2)
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("解密请求失败，状态码: %d, 响应: %s", w2.Code, w2.Body.String())
	}

	// 解析解密响应
	var decryptResponse map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &decryptResponse); err != nil {
		t.Fatalf("解析解密响应失败: %v", err)
	}

	decryptedData := decryptResponse["decrypted_data"].(map[string]interface{})

	// 验证解密结果与原始数据一致
	if !deepEqual(originalPayload, decryptedData) {
		t.Error("解密后的数据与原始数据不一致")
		t.Logf("原始数据: %+v", originalPayload)
		t.Logf("解密数据: %+v", decryptedData)
	}
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// 辅助函数：深度比较两个interface{}
func deepEqual(a, b interface{}) bool {
	aBytes, _ := json.Marshal(a)
	bBytes, _ := json.Marshal(b)
	return string(aBytes) == string(bBytes)
}
