package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"strings"
)

// AnonymizationStrategy 匿名化策略
type AnonymizationStrategy string

const (
	StrategyMapCode        AnonymizationStrategy = "MAP_CODE"
	StrategyTransform      AnonymizationStrategy = "TRANSFORM"
	StrategyMapPlaceholder AnonymizationStrategy = "MAP_PLACEHOLDER"
	StrategyPassthrough    AnonymizationStrategy = "PASSTHROUGH"
)

// AnonymizationRule 匿名化规则
type AnonymizationRule struct {
	Strategy       AnonymizationStrategy  `json:"strategy"`
	StrategyParams map[string]interface{} `json:"strategy_params,omitempty"`
	AppliesTo      struct {
		Type   string        `json:"type"`
		Values []interface{} `json:"values"`
	} `json:"applies_to"`
}

// AnonymizationRequest 匿名化请求
type AnonymizationRequest struct {
	Payload            interface{}         `json:"payload"`
	AnonymizationRules []AnonymizationRule `json:"anonymization_rules"`
}

// AnonymizationResponse 匿名化响应
type AnonymizationResponse struct {
	AnonymizedPayload interface{}            `json:"anonymized_payload"`
	MappingsToStore   map[string]interface{} `json:"mappings_to_store"`
}

// Anonymizer 匿名化器
type Anonymizer struct {
	rules                     []AnonymizationRule
	categoricalMappings       map[string]map[string]string // type -> (code -> original)
	metricPlaceholderMappings map[string]interface{}       // placeholder -> original
	valueToCode               map[string]string            // value -> code (用于确保一致性)
	typeCounters              map[string]int               // type -> counter
	placeholderCounter        int
}

// NewAnonymizer 创建新的匿名化器
func NewAnonymizer(rules []AnonymizationRule) *Anonymizer {
	return &Anonymizer{
		rules:                     rules,
		categoricalMappings:       make(map[string]map[string]string),
		metricPlaceholderMappings: make(map[string]interface{}),
		valueToCode:               make(map[string]string),
		typeCounters:              make(map[string]int),
		placeholderCounter:        0,
	}
}

// Anonymize 执行匿名化
func (a *Anonymizer) Anonymize(payload interface{}) (interface{}, error) {
	result := a.anonymizeValue(payload)
	return result, nil
}

// GetMappings 获取映射表
func (a *Anonymizer) GetMappings() map[string]interface{} {
	mappings := make(map[string]interface{})

	if len(a.categoricalMappings) > 0 {
		mappings["categorical_mappings"] = a.categoricalMappings
	}

	if len(a.metricPlaceholderMappings) > 0 {
		mappings["metric_placeholder_mappings"] = a.metricPlaceholderMappings
	}

	return mappings
}

// anonymizeValue 递归匿名化值
func (a *Anonymizer) anonymizeValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.Map:
		return a.anonymizeMap(value.(map[string]interface{}))
	case reflect.Slice:
		return a.anonymizeSlice(value.([]interface{}))
	case reflect.String:
		return a.anonymizeString(value.(string))
	case reflect.Float64, reflect.Int, reflect.Int64:
		return a.anonymizeNumber(value)
	default:
		return value
	}
}

// anonymizeMap 匿名化map
func (a *Anonymizer) anonymizeMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = a.anonymizeValue(v)
	}
	return result
}

// anonymizeSlice 匿名化切片
func (a *Anonymizer) anonymizeSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[i] = a.anonymizeValue(v)
	}
	return result
}

// anonymizeString 匿名化字符串
func (a *Anonymizer) anonymizeString(s string) string {
	result := s

	// 收集所有需要替换的字符串值及其规则
	type replacement struct {
		value string
		rule  AnonymizationRule
	}
	var replacements []replacement

	for _, rule := range a.rules {
		for _, val := range rule.AppliesTo.Values {
			if valStr, ok := val.(string); ok {
				replacements = append(replacements, replacement{value: valStr, rule: rule})
			}
		}
	}

	// 按值的长度降序排序，优先替换长的值（避免短值替换破坏长值的匹配）
	for i := 0; i < len(replacements); i++ {
		for j := i + 1; j < len(replacements); j++ {
			if len(replacements[i].value) < len(replacements[j].value) {
				replacements[i], replacements[j] = replacements[j], replacements[i]
			}
		}
	}

	// 执行替换
	for _, repl := range replacements {
		if strings.Contains(result, repl.value) {
			code := a.applyStrategy(repl.value, repl.rule)
			result = strings.ReplaceAll(result, repl.value, code)
		}
	}

	return result
}

// anonymizeNumber 匿名化数字
func (a *Anonymizer) anonymizeNumber(n interface{}) interface{} {
	// 查找适用的规则
	for _, rule := range a.rules {
		for _, val := range rule.AppliesTo.Values {
			if compareNumbers(val, n) {
				return a.applyStrategyToNumber(n, rule)
			}
		}
	}
	return n
}

// applyStrategy 应用策略到字符串
func (a *Anonymizer) applyStrategy(value string, rule AnonymizationRule) string {
	switch rule.Strategy {
	case StrategyMapCode:
		return a.generateMapCode(value, rule.AppliesTo.Type)
	case StrategyPassthrough:
		return value
	default:
		return value
	}
}

// applyStrategyToNumber 应用策略到数字
func (a *Anonymizer) applyStrategyToNumber(value interface{}, rule AnonymizationRule) interface{} {
	switch rule.Strategy {
	case StrategyTransform:
		return a.transformNumber(value, rule.StrategyParams)
	case StrategyMapPlaceholder:
		return a.generatePlaceholder(value, rule.AppliesTo.Type)
	case StrategyPassthrough:
		return value
	default:
		return value
	}
}

// generateMapCode 生成映射编码
func (a *Anonymizer) generateMapCode(value string, dataType string) string {
	// 检查是否已经生成过编码
	key := dataType + ":" + value
	if code, exists := a.valueToCode[key]; exists {
		return code
	}

	// 生成新编码（不带大括号）
	if _, exists := a.categoricalMappings[dataType]; !exists {
		a.categoricalMappings[dataType] = make(map[string]string)
	}

	a.typeCounters[dataType]++
	randomSuffix := generateRandomHex(4)
	code := fmt.Sprintf("%s_%s", dataType, randomSuffix)

	// 存储映射（不包括大括号）
	a.categoricalMappings[dataType][code] = value
	a.valueToCode[key] = code

	return code
}

// generatePlaceholder 生成占位符
func (a *Anonymizer) generatePlaceholder(value interface{}, dataType string) string {
	// 检查是否已经生成过占位符
	valueStr := fmt.Sprintf("%v", value)
	key := dataType + ":" + valueStr
	if code, exists := a.valueToCode[key]; exists {
		return code
	}

	// 生成新占位符（不带大括号）
	a.placeholderCounter++
	placeholder := fmt.Sprintf("%s_plc_%d", dataType, a.placeholderCounter)

	// 存储映射（不包括大括号）
	a.metricPlaceholderMappings[placeholder] = value
	a.valueToCode[key] = placeholder

	return placeholder
}

// transformNumber 转换数字（加噪）
func (a *Anonymizer) transformNumber(value interface{}, params map[string]interface{}) float64 {
	var num float64

	switch v := value.(type) {
	case float64:
		num = v
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	default:
		return 0
	}

	noiseLevel := 0.05 // 默认5%
	if params != nil {
		if nl, ok := params["noise_level"].(float64); ok {
			noiseLevel = nl
		}
	}

	// 生成[-1, 1]之间的随机数
	randomFactor := (randFloat() * 2) - 1
	noise := num * noiseLevel * randomFactor

	return num + noise
}

// compareNumbers 比较两个数字是否相等
func compareNumbers(a, b interface{}) bool {
	aFloat := toFloat64(a)
	bFloat := toFloat64(b)
	return math.Abs(aFloat-bFloat) < 0.0001
}

// toFloat64 转换为float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

// generateRandomHex 生成随机十六进制字符串
func generateRandomHex(length int) string {
	bytes := make([]byte, length/2+1)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// randFloat 生成[0,1)之间的随机浮点数
func randFloat() float64 {
	bytes := make([]byte, 8)
	rand.Read(bytes)

	// 将字节转换为uint64
	var n uint64
	for i := 0; i < 8; i++ {
		n = (n << 8) | uint64(bytes[i])
	}

	// 归一化到[0,1)
	return float64(n) / float64(math.MaxUint64)
}
