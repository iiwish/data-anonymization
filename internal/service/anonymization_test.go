package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnonymizer_MapCode(t *testing.T) {
	rules := []AnonymizationRule{
		{
			Strategy: StrategyMapCode,
			AppliesTo: struct {
				Type   string        `json:"type"`
				Values []interface{} `json:"values"`
			}{
				Type:   "REGION",
				Values: []interface{}{"华东", "华北"},
			},
		},
	}

	anonymizer := NewAnonymizer(rules)

	// 测试字符串匿名化
	result1 := anonymizer.anonymizeString("华东")
	result2 := anonymizer.anonymizeString("华东")
	result3 := anonymizer.anonymizeString("华北")

	// 同一个值应该映射到同一个编码
	if result1 != result2 {
		t.Errorf("同一个值应该映射到同一个编码，但得到 %s 和 %s", result1, result2)
	}

	// 不同的值应该映射到不同的编码
	if result1 == result3 {
		t.Errorf("不同的值应该映射到不同的编码，但都得到 %s", result1)
	}

	// 检查编码格式
	if len(result1) < 7 {
		t.Errorf("编码格式不正确: %s", result1)
	}

	// 检查映射表
	mappings := anonymizer.GetMappings()
	catMappings, ok := mappings["categorical_mappings"].(map[string]map[string]string)
	if !ok {
		t.Error("映射表格式不正确")
	}

	regionMappings, ok := catMappings["REGION"]
	if !ok {
		t.Error("缺少REGION类型的映射")
	}

	if regionMappings[result1] != "华东" {
		t.Errorf("映射表不正确，期望 '华东'，得到 '%s'", regionMappings[result1])
	}
}

func TestAnonymizer_Transform(t *testing.T) {
	rules := []AnonymizationRule{
		{
			Strategy: StrategyTransform,
			StrategyParams: map[string]interface{}{
				"noise_level": 0.05,
			},
			AppliesTo: struct {
				Type   string        `json:"type"`
				Values []interface{} `json:"values"`
			}{
				Type:   "REVENUE",
				Values: []interface{}{1500000.0},
			},
		},
	}

	anonymizer := NewAnonymizer(rules)

	// 测试数字转换
	result := anonymizer.anonymizeNumber(1500000.0)

	resultFloat, ok := result.(float64)
	if !ok {
		t.Error("转换结果应该是float64类型")
	}

	// 验证加噪范围（±5%）
	minExpected := 1500000.0 * 0.95
	maxExpected := 1500000.0 * 1.05

	if resultFloat < minExpected || resultFloat > maxExpected {
		t.Errorf("转换后的值 %f 超出预期范围 [%f, %f]", resultFloat, minExpected, maxExpected)
	}
}

func TestAnonymizer_MapPlaceholder(t *testing.T) {
	rules := []AnonymizationRule{
		{
			Strategy: StrategyMapPlaceholder,
			AppliesTo: struct {
				Type   string        `json:"type"`
				Values []interface{} `json:"values"`
			}{
				Type:   "USER_COUNT",
				Values: []interface{}{12000.0, 8500.0},
			},
		},
	}

	anonymizer := NewAnonymizer(rules)

	// 测试占位符生成
	result1 := anonymizer.anonymizeNumber(12000.0)
	result2 := anonymizer.anonymizeNumber(12000.0)
	result3 := anonymizer.anonymizeNumber(8500.0)

	// 同一个值应该生成同一个占位符
	if result1 != result2 {
		t.Errorf("同一个值应该生成同一个占位符，但得到 %s 和 %s", result1, result2)
	}

	// 不同的值应该生成不同的占位符
	if result1 == result3 {
		t.Errorf("不同的值应该生成不同的占位符，但都得到 %s", result1)
	}

	// 检查映射表
	mappings := anonymizer.GetMappings()
	metricMappings, ok := mappings["metric_placeholder_mappings"].(map[string]interface{})
	if !ok {
		t.Error("映射表格式不正确")
	}

	placeholder1, ok := result1.(string)
	if !ok {
		t.Error("占位符应该是字符串类型")
	}

	if metricMappings[placeholder1] != 12000.0 {
		t.Errorf("映射表不正确，期望 12000，得到 %v", metricMappings[placeholder1])
	}
}

func TestAnonymizer_ComplexPayload(t *testing.T) {
	rules := []AnonymizationRule{
		{
			Strategy: StrategyMapCode,
			AppliesTo: struct {
				Type   string        `json:"type"`
				Values []interface{} `json:"values"`
			}{
				Type:   "REGION",
				Values: []interface{}{"华东", "华北"},
			},
		},
		{
			Strategy: StrategyMapCode,
			AppliesTo: struct {
				Type   string        `json:"type"`
				Values []interface{} `json:"values"`
			}{
				Type:   "PRODUCT",
				Values: []interface{}{"手机", "电脑"},
			},
		},
	}

	anonymizer := NewAnonymizer(rules)

	// 创建复杂的payload
	payload := map[string]interface{}{
		"metadata": map[string]interface{}{
			"report_name": "Q3 Sales Analysis for 华东",
			"requester":   "user123",
		},
		"data_table": []interface{}{
			map[string]interface{}{
				"区域":   "华东",
				"核心产品": "手机",
				"季度收入": 1500000.0,
			},
			map[string]interface{}{
				"区域":   "华北",
				"核心产品": "电脑",
				"季度收入": 950000.0,
			},
		},
	}

	result, err := anonymizer.Anonymize(payload)
	if err != nil {
		t.Fatalf("匿名化失败: %v", err)
	}

	// 将结果转换为map
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

	// report_name中的"华东"应该被替换
	if reportName == "Q3 Sales Analysis for 华东" {
		t.Error("华东应该被匿名化")
	}

	// 检查data_table
	dataTable, ok := resultMap["data_table"].([]interface{})
	if !ok {
		t.Fatal("data_table格式不正确")
	}

	if len(dataTable) != 2 {
		t.Errorf("data_table应该有2个元素，但有 %d 个", len(dataTable))
	}

	firstRow, ok := dataTable[0].(map[string]interface{})
	if !ok {
		t.Fatal("data_table第一行格式不正确")
	}

	region, ok := firstRow["区域"].(string)
	if !ok {
		t.Fatal("区域应该是字符串")
	}

	if region == "华东" {
		t.Error("区域应该被匿名化")
	}

	// 验证映射表
	mappings := anonymizer.GetMappings()
	catMappings, ok := mappings["categorical_mappings"].(map[string]map[string]string)
	if !ok {
		t.Fatal("映射表格式不正确")
	}

	if len(catMappings) != 2 {
		t.Errorf("应该有2种类型的映射，但有 %d 种", len(catMappings))
	}

	if _, ok := catMappings["REGION"]; !ok {
		t.Error("缺少REGION映射")
	}

	if _, ok := catMappings["PRODUCT"]; !ok {
		t.Error("缺少PRODUCT映射")
	}
}

func TestAnonymizer_Passthrough(t *testing.T) {
	rules := []AnonymizationRule{
		{
			Strategy: StrategyPassthrough,
			AppliesTo: struct {
				Type   string        `json:"type"`
				Values []interface{} `json:"values"`
			}{
				Type:   "GROWTH_RATE",
				Values: []interface{}{"12.5%", "-3.2%"},
			},
		},
	}

	anonymizer := NewAnonymizer(rules)

	// 测试passthrough策略
	result := anonymizer.anonymizeString("12.5%")

	if result != "12.5%" {
		t.Errorf("Passthrough策略应该保持原值，期望 '12.5%%'，得到 '%s'", result)
	}
}

func TestAnonymizationRequest_JSON(t *testing.T) {
	// 测试JSON序列化和反序列化
	jsonStr := `{
		"payload": {
			"test": "value"
		},
		"anonymization_rules": [
			{
				"strategy": "MAP_CODE",
				"applies_to": {
					"type": "TEST",
					"values": ["value"]
				}
			}
		]
	}`

	var req AnonymizationRequest
	err := json.Unmarshal([]byte(jsonStr), &req)
	if err != nil {
		t.Fatalf("JSON反序列化失败: %v", err)
	}

	if len(req.AnonymizationRules) != 1 {
		t.Errorf("规则数量不正确，期望 1，得到 %d", len(req.AnonymizationRules))
	}

	if req.AnonymizationRules[0].Strategy != StrategyMapCode {
		t.Errorf("策略不正确，期望 'MAP_CODE'，得到 '%s'", req.AnonymizationRules[0].Strategy)
	}
}

func TestAnonymizer_EdgeCases(t *testing.T) {
	// 测试空规则
	rules := []AnonymizationRule{}
	anonymizer := NewAnonymizer(rules)

	payload := map[string]interface{}{
		"test": "value",
	}

	result, err := anonymizer.Anonymize(payload)
	if err != nil {
		t.Fatalf("匿名化失败: %v", err)
	}

	// 没有规则时应该返回原值
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("结果应该是map类型")
	}

	if resultMap["test"] != "value" {
		t.Errorf("没有规则时应该保持原值，期望 'value'，得到 '%v'", resultMap["test"])
	}

	// 测试nil payload
	result2, err := anonymizer.Anonymize(nil)
	if err != nil {
		t.Fatalf("匿名化nil失败: %v", err)
	}

	if result2 != nil {
		t.Errorf("nil payload应该返回nil，得到 %v", result2)
	}
}

func TestAnonymizer_ComplexNestedStructure(t *testing.T) {
	rules := []AnonymizationRule{
		{
			Strategy: StrategyMapCode,
			AppliesTo: struct {
				Type   string        `json:"type"`
				Values []interface{} `json:"values"`
			}{
				Type:   "REGION",
				Values: []interface{}{"华东", "华北"},
			},
		},
	}

	anonymizer := NewAnonymizer(rules)

	// 创建复杂的嵌套结构
	payload := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": []interface{}{
					map[string]interface{}{
						"region": "华东",
						"data":   []interface{}{"华东", "其他"},
					},
					map[string]interface{}{
						"region": "华北",
						"data":   []interface{}{"华北", "其他"},
					},
				},
			},
		},
	}

	result, err := anonymizer.Anonymize(payload)
	if err != nil {
		t.Fatalf("匿名化失败: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("结果应该是map类型")
	}

	// 验证嵌套结构中的值已被替换
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

	firstItem, ok := level3[0].(map[string]interface{})
	if !ok {
		t.Fatal("level3第一个元素格式不正确")
	}

	region, ok := firstItem["region"].(string)
	if !ok {
		t.Fatal("region应该是字符串")
	}

	if region == "华东" {
		t.Error("嵌套结构中的华东应该被匿名化")
	}

	// 验证映射表
	mappings := anonymizer.GetMappings()
	catMappings, ok := mappings["categorical_mappings"].(map[string]map[string]string)
	if !ok {
		t.Fatal("映射表格式不正确")
	}

	if len(catMappings["REGION"]) != 2 {
		t.Errorf("应该有2个REGION映射，但有 %d 个", len(catMappings["REGION"]))
	}
}

func TestAnonymizer_EncodingFormat(t *testing.T) {
	rules := []AnonymizationRule{
		{
			Strategy: StrategyMapCode,
			AppliesTo: struct {
				Type   string        `json:"type"`
				Values []interface{} `json:"values"`
			}{
				Type:   "TEST",
				Values: []interface{}{"test_value"},
			},
		},
	}

	anonymizer := NewAnonymizer(rules)

	result := anonymizer.anonymizeString("test_value")

	// 验证编码格式：应该不带大括号
	if strings.Contains(result, "{") || strings.Contains(result, "}") {
		t.Errorf("编码不应该包含大括号，得到: %s", result)
	}

	// 验证编码格式：应该以类型开头
	if !strings.HasPrefix(result, "TEST_") {
		t.Errorf("编码应该以类型开头，得到: %s", result)
	}

	// 验证编码长度
	if len(result) < 8 {
		t.Errorf("编码长度太短: %s", result)
	}
}

func TestAnonymizer_StringReplacementOrder(t *testing.T) {
	rules := []AnonymizationRule{
		{
			Strategy: StrategyMapCode,
			AppliesTo: struct {
				Type   string        `json:"type"`
				Values []interface{} `json:"values"`
			}{
				Type:   "SHORT",
				Values: []interface{}{"华东"},
			},
		},
		{
			Strategy: StrategyMapCode,
			AppliesTo: struct {
				Type   string        `json:"type"`
				Values []interface{} `json:"values"`
			}{
				Type:   "LONG",
				Values: []interface{}{"华东地区"},
			},
		},
	}

	anonymizer := NewAnonymizer(rules)

	// 测试替换顺序：长值应该先被替换，避免短值替换破坏长值
	text := "华东地区的华东"
	result := anonymizer.anonymizeString(text)

	// 验证长值"华东地区"被替换
	if strings.Contains(result, "华东地区") {
		t.Error("长值'华东地区'应该被替换")
	}

	// 验证短值"华东"被替换
	if strings.Contains(result, "华东") {
		t.Error("短值'华东'应该被替换")
	}

	// 验证结果中不包含原始值
	if result == text {
		t.Error("文本应该被匿名化")
	}
}

func TestAnonymizer_NumberTypes(t *testing.T) {
	rules := []AnonymizationRule{
		{
			Strategy: StrategyTransform,
			StrategyParams: map[string]interface{}{
				"noise_level": 0.05,
			},
			AppliesTo: struct {
				Type   string        `json:"type"`
				Values []interface{} `json:"values"`
			}{
				Type:   "REVENUE",
				Values: []interface{}{1000.0, 2000},
			},
		},
	}

	anonymizer := NewAnonymizer(rules)

	// 测试float64
	result1 := anonymizer.anonymizeNumber(1000.0)
	result1Float, ok := result1.(float64)
	if !ok {
		t.Error("float64转换结果应该是float64类型")
	}

	// 测试int
	result2 := anonymizer.anonymizeNumber(2000)
	result2Float, ok := result2.(float64)
	if !ok {
		t.Error("int转换结果应该是float64类型")
	}

	// 验证加噪范围
	if result1Float < 950 || result1Float > 1050 {
		t.Errorf("转换后的值 %f 超出预期范围 [950, 1050]", result1Float)
	}

	if result2Float < 1900 || result2Float > 2100 {
		t.Errorf("转换后的值 %f 超出预期范围 [1900, 2100]", result2Float)
	}
}
