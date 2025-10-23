package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// SystemConfig 表示单个系统的配置信息
type SystemConfig struct {
	SystemID     string `json:"system_id"`
	SharedSecret string `json:"shared_secret"`
	Description  string `json:"description"`
}

// ServerConfig 表示服务器配置
type ServerConfig struct {
	Port                   int    `json:"port"`
	LogFile                string `json:"log_file"`
	TimestampWindowSeconds int    `json:"timestamp_window_seconds"`
}

// Config 表示完整的应用配置
type Config struct {
	Systems []SystemConfig `json:"systems"`
	Server  ServerConfig   `json:"server"`
}

// systemMap 缓存system_id到配置的映射
var systemMap map[string]*SystemConfig

// Load 从指定路径加载配置文件
func Load(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("无法打开配置文件: %w", err)
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 构建系统映射
	systemMap = make(map[string]*SystemConfig)
	for i := range cfg.Systems {
		systemMap[cfg.Systems[i].SystemID] = &cfg.Systems[i]
	}

	return &cfg, nil
}

// validate 验证配置的有效性
func (c *Config) validate() error {
	if len(c.Systems) == 0 {
		return fmt.Errorf("至少需要配置一个系统")
	}

	// 检查system_id是否重复
	seen := make(map[string]bool)
	for _, sys := range c.Systems {
		if sys.SystemID == "" {
			return fmt.Errorf("system_id不能为空")
		}
		if sys.SharedSecret == "" {
			return fmt.Errorf("system '%s' 的shared_secret不能为空", sys.SystemID)
		}
		if seen[sys.SystemID] {
			return fmt.Errorf("system_id '%s' 重复", sys.SystemID)
		}
		seen[sys.SystemID] = true
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("无效的端口号: %d", c.Server.Port)
	}

	if c.Server.TimestampWindowSeconds <= 0 {
		c.Server.TimestampWindowSeconds = 300 // 默认5分钟
	}

	return nil
}

// GetSystemConfig 根据system_id获取系统配置
func GetSystemConfig(systemID string) (*SystemConfig, bool) {
	sys, ok := systemMap[systemID]
	return sys, ok
}
