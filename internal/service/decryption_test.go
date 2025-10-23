package service

import (
	"encoding/json"
	"testing"
)

func TestDecryptor_DecryptString(t *testing.T) {
	mappings := map[string]interface{}{
		"categorical_mappings": map[string]interface{}{
			"REGION": map[string]interface{}{
				"REGION_a3f5": "华东",
				"REGION_b1e9": "华北",
			},
			"PRODUCT": map[string]interface{}{
				"PRODUCT_c8b1": "手机",
				"PRODUCT_d2a7": "电脑",
			},
		},
		"metric_placeholder_mappings": map[string]interface{}{
			"USER_COUNT_plc_1": 12000.0,
			"USER_COUNT_plc_2": 8500.0,
		},
	}

	decryptor := NewDecryptor(mappings)

	// 测试纯文本解密
	text := "分析显示，REGION_a3f5 区域的 PRODUCT_c8b1 表现最佳，活跃用户数为 USER_COUNT_plc_1。"
	result := decryptor.Decrypt(text)

	resultStr, ok := result.(string)
	if !ok {
		t.Fatal("结果应该是字符串类型")
	}

	expectedSubstrings := []string{"华东", "手机"}
	for _, expected := range expectedSubstrings {
		if !contains(resultStr, expected) {
			t.Errorf("解密后的文本应该包含 '%s'，但实际是: %s", expected, resultStr)
		}
	}

	// 验证编码已被替换
	unexpectedSubstrings := []string{"REGION_a3f5", "PRODUCT_c8b1"}
	for _, unexpected := range unexpectedSubstrings {
		if contains(resultStr, unexpected) {
			t.Errorf("解密后的文本不应该包含 '%s'，但实际是: %s", unexpected, resultStr)
		}
	}
}

func TestDecryptor_DecryptJSON(t *testing.T) {
	mappings := map[string]interface{}{
		"categorical_mappings": map[string]interface{}{
			"REGION": map[string]interface{}{
				"REGION_a3f5": "华东",
				"REGION_b1e9": "华北",
			},
			"PRODUCT": map[string]interface{}{
				"PRODUCT_c8b1": "手机",
			},
		},
		"metric_placeholder_mappings": map[string]interface{}{
			"USER_COUNT_plc_1": 12000.0,
		},
	}

	decryptor := NewDecryptor(mappings)

	// 创建包含匿名编码的JSON对象
	data := map[string]interface{}{
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
	}

	result := decryptor.Decrypt(data)

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("结果应该是map类型")
	}

	// 检查summary
	summary, ok := resultMap["summary"].(string)
	if !ok {
		t.Fatal("summary应该是字符串")
	}

	if !contains(summary, "华东") {
		t.Errorf("summary应该包含 '华东'，但实际是: %s", summary)
	}

	if !contains(summary, "手机") {
		t.Errorf("summary应该包含 '手机'，但实际是: %s", summary)
	}

	// 检查key_findings
	findings, ok := resultMap["key_findings"].([]interface{})
	if !ok {
		t.Fatal("key_findings应该是数组")
	}

	if len(findings) != 2 {
		t.Fatalf("key_findings应该有2个元素，但有 %d 个", len(findings))
	}

	// 检查第一个finding
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

	// 检查第二个finding（产品编码应该被替换）
	secondFinding, ok := findings[1].(map[string]interface{})
	if !ok {
		t.Fatal("第二个finding应该是map")
	}

	productValue, ok := secondFinding["value"].(string)
	if !ok {
		t.Fatal("value应该是字符串")
	}

	if productValue != "手机" {
		t.Errorf("value应该是 '手机'，但是 '%s'", productValue)
	}
}

func TestDecryptor_ComplexPayload(t *testing.T) {
	mappings := map[string]interface{}{
		"categorical_mappings": map[string]interface{}{
			"REGION": map[string]interface{}{
				"REGION_a3f5": "华东",
				"REGION_b1e9": "华北",
			},
			"PRODUCT": map[string]interface{}{
				"PRODUCT_c8b1": "手机",
				"PRODUCT_d2a7": "电脑",
			},
		},
		"metric_placeholder_mappings": map[string]interface{}{
			"USER_COUNT_plc_1": 12000.0,
			"USER_COUNT_plc_2": 8500.0,
		},
	}

	decryptor := NewDecryptor(mappings)

	// 创建复杂的匿名化payload
	anonymizedData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"report_name": "Q3 Sales Analysis for REGION_a3f5",
			"requester":   "user123",
		},
		"analysis_prompt": "Analyze the following sales data. The previous quarter's top product was PRODUCT_c8b1.",
		"data_table": []interface{}{
			map[string]interface{}{
				"区域":    "REGION_a3f5",
				"核心产品":  "PRODUCT_c8b1",
				"活跃用户数": "USER_COUNT_plc_1",
			},
			map[string]interface{}{
				"区域":    "REGION_b1e9",
				"核心产品":  "PRODUCT_d2a7",
				"活跃用户数": "USER_COUNT_plc_2",
			},
		},
	}

	result := decryptor.Decrypt(anonymizedData)

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("结果应该是map类型")
	}

	// 检查metadata
	metadata, ok := resultMap["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata格式不正确")
	}

	reportName, ok := metadata["report_name"].(string)
	if !ok {
		t.Fatal("report_name应该是字符串")
	}

	if !contains(reportName, "华东") {
		t.Errorf("report_name应该包含 '华东'，但实际是: %s", reportName)
	}

	// 检查analysis_prompt
	prompt, ok := resultMap["analysis_prompt"].(string)
	if !ok {
		t.Fatal("analysis_prompt应该是字符串")
	}

	if !contains(prompt, "手机") {
		t.Errorf("analysis_prompt应该包含 '手机'，但实际是: %s", prompt)
	}

	// 检查data_table
	dataTable, ok := resultMap["data_table"].([]interface{})
	if !ok {
		t.Fatal("data_table格式不正确")
	}

	if len(dataTable) != 2 {
		t.Errorf("data_table应该有2个元素，但有 %d 个", len(dataTable))
	}

	// 检查第一行数据
	firstRow, ok := dataTable[0].(map[string]interface{})
	if !ok {
		t.Fatal("第一行数据格式不正确")
	}

	region, ok := firstRow["区域"].(string)
	if !ok {
		t.Fatal("区域应该是字符串")
	}

	if region != "华东" {
		t.Errorf("区域应该是 '华东'，但是 '%s'", region)
	}

	product, ok := firstRow["核心产品"].(string)
	if !ok {
		t.Fatal("核心产品应该是字符串")
	}

	if product != "手机" {
		t.Errorf("核心产品应该是 '手机'，但是 '%s'", product)
	}

	// 活跃用户数应该被替换为数值
	userCount := firstRow["活跃用户数"]
	if userCount != 12000.0 {
		t.Errorf("活跃用户数应该是 12000，但是 %v", userCount)
	}
}

func TestDecryptor_EmptyMappings(t *testing.T) {
	// 测试空映射表
	mappings := map[string]interface{}{}
	decryptor := NewDecryptor(mappings)

	text := "REGION_a3f5 PRODUCT_c8b1"
	result := decryptor.Decrypt(text)

	resultStr, ok := result.(string)
	if !ok {
		t.Fatal("结果应该是字符串类型")
	}

	// 没有映射，所以应该保持原样
	if resultStr != text {
		t.Errorf("没有映射时应该保持原样，期望 '%s'，得到 '%s'", text, resultStr)
	}
}

func TestDecryptor_PartialMappings(t *testing.T) {
	// 测试部分映射
	mappings := map[string]interface{}{
		"categorical_mappings": map[string]interface{}{
			"REGION": map[string]interface{}{
				"REGION_a3f5": "华东",
			},
		},
	}

	decryptor := NewDecryptor(mappings)

	text := "REGION_a3f5 和 REGION_unknown 的对比"
	result := decryptor.Decrypt(text)

	resultStr, ok := result.(string)
	if !ok {
		t.Fatal("结果应该是字符串类型")
	}

	// REGION_a3f5应该被替换，REGION_unknown应该保持原样
	if !contains(resultStr, "华东") {
		t.Errorf("结果应该包含 '华东'，但实际是: %s", resultStr)
	}

	if !contains(resultStr, "REGION_unknown") {
		t.Errorf("结果应该保留 'REGION_unknown'，但实际是: %s", resultStr)
	}
}

func TestDecryptionRequest_JSON(t *testing.T) {
	// 测试JSON序列化和反序列化
	jsonStr := `{
		"data_with_anonymized_codes": "REGION_a3f5",
		"mappings": {
			"categorical_mappings": {
				"REGION": {
					"REGION_a3f5": "华东"
				}
			}
		}
	}`

	var req DecryptionRequest
	err := json.Unmarshal([]byte(jsonStr), &req)
	if err != nil {
		t.Fatalf("JSON反序列化失败: %v", err)
	}

	dataStr, ok := req.DataWithAnonymizedCodes.(string)
	if !ok {
		t.Fatal("data_with_anonymized_codes应该是字符串")
	}

	if dataStr != "REGION_a3f5" {
		t.Errorf("data_with_anonymized_codes不正确，期望 'REGION_a3f5'，得到 '%s'", dataStr)
	}

	if req.Mappings == nil {
		t.Fatal("mappings不应该为空")
	}
}

func TestDecryptor_NestedStructures(t *testing.T) {
	mappings := map[string]interface{}{
		"categorical_mappings": map[string]interface{}{
			"REGION": map[string]interface{}{
				"REGION_a3f5": "华东",
			},
		},
	}

	decryptor := NewDecryptor(mappings)

	// 测试深层嵌套的结构
	data := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": []interface{}{
					map[string]interface{}{
						"value": "REGION_a3f5",
					},
				},
			},
		},
	}

	result := decryptor.Decrypt(data)

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("结果应该是map类型")
	}

	// 导航到深层嵌套的值
	level1, ok := resultMap["level1"].(map[string]interface{})
	if !ok {
		t.Fatal("level1格式不正确")
	}

	level2, ok := level1["level2"].(map[string]interface{})
	if !ok {
		t.Fatal("level2格式不正确")
	}

	level3, ok := level2["level3"].([]interface{})
	if !ok {
		t.Fatal("level3格式不正确")
	}

	item, ok := level3[0].(map[string]interface{})
	if !ok {
		t.Fatal("level3第一个元素格式不正确")
	}

	value, ok := item["value"].(string)
	if !ok {
		t.Fatal("value应该是字符串")
	}

	if value != "华东" {
		t.Errorf("value应该是 '华东'，但是 '%s'", value)
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
