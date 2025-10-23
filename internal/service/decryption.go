package service

import (
	"fmt"
	"reflect"
	"strings"
)

// DecryptionRequest 解密请求
type DecryptionRequest struct {
	DataWithAnonymizedCodes interface{}            `json:"data_with_anonymized_codes"`
	Mappings                map[string]interface{} `json:"mappings"`
}

// DecryptionResponse 解密响应
type DecryptionResponse struct {
	DecryptedData interface{} `json:"decrypted_data"`
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
func (d *Decryptor) Decrypt(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	// 判断数据类型
	v := reflect.ValueOf(data)

	switch v.Kind() {
	case reflect.String:
		// 纯文本字符串，进行全局替换
		return d.decryptString(data.(string))
	case reflect.Map:
		// JSON对象，递归处理
		return d.decryptValue(data)
	case reflect.Slice:
		// 数组，递归处理
		return d.decryptValue(data)
	default:
		return data
	}
}

// decryptValue 递归解密值
func (d *Decryptor) decryptValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.Map:
		return d.decryptMap(value.(map[string]interface{}))
	case reflect.Slice:
		return d.decryptSlice(value.([]interface{}))
	case reflect.String:
		return d.decryptStringValue(value.(string))
	default:
		return value
	}
}

// decryptMap 解密map
func (d *Decryptor) decryptMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = d.decryptValue(v)
	}
	return result
}

// decryptSlice 解密切片
func (d *Decryptor) decryptSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[i] = d.decryptValue(v)
	}
	return result
}

// decryptStringValue 解密字符串值（精确替换）
func (d *Decryptor) decryptStringValue(s string) interface{} {
	// 首先检查是否是占位符（完全匹配）
	if value, ok := d.metricPlaceholderMappings[s]; ok {
		return value
	}

	// 然后检查是否是编码（完全匹配）
	if value, ok := d.codeToValue[s]; ok {
		return value
	}

	// 如果不是完全匹配，进行文本内替换
	return d.decryptString(s)
}

// decryptString 解密纯文本字符串（全局替换）
func (d *Decryptor) decryptString(text string) string {
	result := text

	// 替换所有分类编码
	for code, value := range d.codeToValue {
		result = strings.ReplaceAll(result, code, value)
	}

	// 替换所有占位符
	for placeholder, value := range d.metricPlaceholderMappings {
		// 将数值转换为字符串
		valueStr := formatValue(value)
		result = strings.ReplaceAll(result, placeholder, valueStr)
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
