package service

import (
	"fmt"
	"strings"
)

// DecryptionRequest 解密请求
type DecryptionRequest struct {
	DataWithAnonymizedCodes string                 `json:"data_with_anonymized_codes"`
	Mappings                map[string]interface{} `json:"mappings"`
}

// DecryptionResponse 解密响应
type DecryptionResponse struct {
	DecryptedData string `json:"decrypted_data"`
}

// Decryptor 解密器
type Decryptor struct {
	categoricalMappings       map[string]map[string]string // type -> (code -> original)
	metricPlaceholderMappings map[string]interface{}       // placeholder -> original
	codeToValue               map[string]string            // 所有code到原始值的映射（用于文本替换）
}

// NewDecryptor 创建新的解密器
func NewDecryptor(mappings map[string]interface{}) *Decryptor {
	d := &Decryptor{
		categoricalMappings:       make(map[string]map[string]string),
		metricPlaceholderMappings: make(map[string]interface{}),
		codeToValue:               make(map[string]string),
	}

	// 解析categorical_mappings
	if catMappings, ok := mappings["categorical_mappings"].(map[string]interface{}); ok {
		for typeKey, typeMap := range catMappings {
			d.categoricalMappings[typeKey] = make(map[string]string)
			if tm, ok := typeMap.(map[string]interface{}); ok {
				for code, value := range tm {
					if valStr, ok := value.(string); ok {
						d.categoricalMappings[typeKey][code] = valStr
						d.codeToValue[code] = valStr
					}
				}
			}
		}
	}

	// 解析metric_placeholder_mappings
	if metricMappings, ok := mappings["metric_placeholder_mappings"].(map[string]interface{}); ok {
		for placeholder, value := range metricMappings {
			d.metricPlaceholderMappings[placeholder] = value
		}
	}

	return d
}

// Decrypt 执行解密
func (d *Decryptor) Decrypt(text string) string {
	// 只处理纯文本字符串
	return d.decryptString(text)
}

// decryptString 解密纯文本字符串（全局替换）
func (d *Decryptor) decryptString(text string) string {
	result := text

	// 替换所有分类编码（大括号格式）
	for code, value := range d.codeToValue {
		encodedCode := "{" + code + "}"
		result = strings.ReplaceAll(result, encodedCode, value)
	}

	// 替换所有占位符（大括号格式）
	for placeholder, value := range d.metricPlaceholderMappings {
		encodedPlaceholder := "{" + placeholder + "}"
		// 将数值转换为字符串
		valueStr := formatValue(value)
		result = strings.ReplaceAll(result, encodedPlaceholder, valueStr)
	}

	return result
}

// formatValue 格式化值为字符串
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		// 如果是整数，去掉小数点
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", value)
	}
}
