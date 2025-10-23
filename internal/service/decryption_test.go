package service

import (
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
	text := "分析显示，{REGION_a3f5} 区域的 {PRODUCT_c8b1} 表现最佳，活跃用户数为 {USER_COUNT_plc_1}。"
	resultStr := decryptor.Decrypt(text)

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

func TestDecryptor_EmptyMappings(t *testing.T) {
	// 测试空映射表
	mappings := map[string]interface{}{}
	decryptor := NewDecryptor(mappings)

	text := "{REGION_a3f5} {PRODUCT_c8b1}"
	resultStr := decryptor.Decrypt(text)

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

	text := "{REGION_a3f5} 和 {REGION_unknown} 的对比"
	resultStr := decryptor.Decrypt(text)

	// REGION_a3f5应该被替换，REGION_unknown应该保持原样
	if !contains(resultStr, "华东") {
		t.Errorf("结果应该包含 '华东'，但实际是: %s", resultStr)
	}

	if !contains(resultStr, "REGION_unknown") {
		t.Errorf("结果应该保留 'REGION_unknown'，但实际是: %s", resultStr)
	}
}

func TestDecryptor_BraceFormat(t *testing.T) {
	mappings := map[string]interface{}{
		"categorical_mappings": map[string]interface{}{
			"REGION": map[string]interface{}{
				"REGION_a3f5": "华东",
				"REGION_b1e9": "华北",
			},
		},
		"metric_placeholder_mappings": map[string]interface{}{
			"USER_COUNT_plc_1": 12000.0,
		},
	}

	decryptor := NewDecryptor(mappings)

	// 测试带大括号的编码应该被替换
	text := "分析显示，{REGION_a3f5} 区域的活跃用户数为 {USER_COUNT_plc_1}。"
	resultStr := decryptor.Decrypt(text)

	// 验证编码已被替换
	if contains(resultStr, "{REGION_a3f5}") {
		t.Errorf("带大括号的编码应该被替换，但实际是: %s", resultStr)
	}

	if contains(resultStr, "{USER_COUNT_plc_1}") {
		t.Errorf("带大括号的占位符应该被替换，但实际是: %s", resultStr)
	}

	// 验证原始值已正确替换
	if !contains(resultStr, "华东") {
		t.Errorf("结果应该包含 '华东'，但实际是: %s", resultStr)
	}

	if !contains(resultStr, "12000") {
		t.Errorf("结果应该包含 '12000'，但实际是: %s", resultStr)
	}
}

func TestDecryptor_NoBraceNoReplace(t *testing.T) {
	mappings := map[string]interface{}{
		"categorical_mappings": map[string]interface{}{
			"REGION": map[string]interface{}{
				"REGION_a3f5": "华东",
			},
		},
	}

	decryptor := NewDecryptor(mappings)

	// 测试不带大括号的编码不应该被替换
	text := "分析显示，REGION_a3f5 区域表现良好。"
	resultStr := decryptor.Decrypt(text)

	// 不带大括号的编码应该保持原样
	if !contains(resultStr, "REGION_a3f5") {
		t.Errorf("不带大括号的编码应该保持原样，但实际是: %s", resultStr)
	}

	if contains(resultStr, "华东") {
		t.Errorf("不带大括号的编码不应该被替换，但实际是: %s", resultStr)
	}
}

func TestDecryptor_MixedBraceAndNoBrace(t *testing.T) {
	mappings := map[string]interface{}{
		"categorical_mappings": map[string]interface{}{
			"REGION": map[string]interface{}{
				"REGION_a3f5": "华东",
			},
		},
	}

	decryptor := NewDecryptor(mappings)

	// 测试混合格式：带大括号的被替换，不带大括号的保持原样
	text := "分析显示，{REGION_a3f5} 区域和 REGION_a3f5 区域表现良好。"
	resultStr := decryptor.Decrypt(text)

	// 带大括号的应该被替换
	if contains(resultStr, "{REGION_a3f5}") {
		t.Errorf("带大括号的编码应该被替换，但实际是: %s", resultStr)
	}

	// 不带大括号的应该保持原样
	if !contains(resultStr, "REGION_a3f5") {
		t.Errorf("不带大括号的编码应该保持原样，但实际是: %s", resultStr)
	}

	// 验证替换结果
	if !contains(resultStr, "华东") {
		t.Errorf("结果应该包含 '华东'，但实际是: %s", resultStr)
	}
}

func TestDecryptor_ComplexText(t *testing.T) {
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
			"REVENUE_plc_1":    1500000.0,
		},
	}

	decryptor := NewDecryptor(mappings)

	// 测试复杂文本场景
	text := `根据数据分析，{REGION_a3f5} 区域的 {PRODUCT_c8b1} 表现最佳，活跃用户数达到 {USER_COUNT_plc_1}，收入为 {REVENUE_plc_1}。
相比之下，{REGION_b1e9} 区域的 {PRODUCT_d2a7} 表现相对较弱。建议重点关注 {REGION_a3f5} 区域的发展。`

	resultStr := decryptor.Decrypt(text)

	// 验证所有编码都被正确替换
	expectedSubstrings := []string{"华东", "手机", "12000", "1500000", "华北", "电脑"}
	for _, expected := range expectedSubstrings {
		if !contains(resultStr, expected) {
			t.Errorf("解密后的文本应该包含 '%s'，但实际是: %s", expected, resultStr)
		}
	}

	// 验证所有编码都被移除
	unexpectedSubstrings := []string{"{REGION_a3f5}", "{REGION_b1e9}", "{PRODUCT_c8b1}", "{PRODUCT_d2a7}", "{USER_COUNT_plc_1}", "{REVENUE_plc_1}"}
	for _, unexpected := range unexpectedSubstrings {
		if contains(resultStr, unexpected) {
			t.Errorf("解密后的文本不应该包含 '%s'，但实际是: %s", unexpected, resultStr)
		}
	}
}

func TestDecryptor_NumberFormatting(t *testing.T) {
	mappings := map[string]interface{}{
		"metric_placeholder_mappings": map[string]interface{}{
			"INTEGER_plc_1": 1000.0,   // 整数
			"FLOAT_plc_1":   1234.567, // 浮点数
			"INT_plc_1":     500,      // int类型
		},
	}

	decryptor := NewDecryptor(mappings)

	text := "整数: {INTEGER_plc_1}, 浮点数: {FLOAT_plc_1}, 整型: {INT_plc_1}"
	resultStr := decryptor.Decrypt(text)

	// 验证整数格式（应该去掉小数点）
	if !contains(resultStr, "1000") {
		t.Errorf("整数应该格式化为 '1000'，但实际是: %s", resultStr)
	}

	// 验证浮点数格式
	if !contains(resultStr, "1234.567000") {
		t.Errorf("浮点数应该格式化为 '1234.567000'，但实际是: %s", resultStr)
	}

	// 验证int类型格式
	if !contains(resultStr, "500") {
		t.Errorf("int类型应该格式化为 '500'，但实际是: %s", resultStr)
	}
}

func TestDecryptor_InvalidMappings(t *testing.T) {
	// 测试无效的映射表结构
	mappings := map[string]interface{}{
		"categorical_mappings":        "invalid_string",          // 应该是map
		"metric_placeholder_mappings": []string{"invalid_array"}, // 应该是map
	}

	decryptor := NewDecryptor(mappings)

	// 应该能正常处理，只是没有替换
	text := "{REGION_a3f5} {USER_COUNT_plc_1}"
	resultStr := decryptor.Decrypt(text)

	// 由于映射表无效，编码应该保持原样
	if resultStr != text {
		t.Errorf("无效映射表时应该保持原样，期望 '%s'，得到 '%s'", text, resultStr)
	}
}

func TestDecryptor_PartialReplacement(t *testing.T) {
	mappings := map[string]interface{}{
		"categorical_mappings": map[string]interface{}{
			"REGION": map[string]interface{}{
				"REGION_a3f5": "华东",
			},
		},
	}

	decryptor := NewDecryptor(mappings)

	// 测试部分替换：只有存在的编码被替换，不存在的保持原样
	text := "{REGION_a3f5} 和 {REGION_unknown} 的对比，以及 {PRODUCT_missing} 产品"
	resultStr := decryptor.Decrypt(text)

	// 验证存在的编码被替换
	if !contains(resultStr, "华东") {
		t.Errorf("结果应该包含 '华东'，但实际是: %s", resultStr)
	}

	// 验证不存在的编码保持原样
	if !contains(resultStr, "{REGION_unknown}") {
		t.Errorf("不存在的编码应该保持原样，但实际是: %s", resultStr)
	}

	if !contains(resultStr, "{PRODUCT_missing}") {
		t.Errorf("不存在的编码应该保持原样，但实际是: %s", resultStr)
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
