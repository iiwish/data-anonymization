# 数据匿名化与解密服务

这是一个通用的数据隐私保护微服务，提供数据匿名化和解密功能，特别适用于需要与第三方AI系统交互的场景。

## 功能特性

- **匿名化服务 (`/v1/anonymize`)**: 将敏感数据替换为安全编码
- **解密服务 (`/v1/decrypt`)**: 将匿名编码还原为原始数据
- **HMAC鉴权**: 基于HMAC-SHA256的安全认证机制
- **多租户支持**: 支持多个系统独立接入
- **文件日志**: 结构化JSON日志记录
- **多种匿名化策略**:
  - MAP_CODE: 映射编码
  - TRANSFORM: 数值加噪
  - MAP_PLACEHOLDER: 占位符映射
  - PASSTHROUGH: 透传

## 技术栈

- Go 1.24.3
- Gin Web Framework
- Viper (配置管理)

## 快速开始

### 1. 安装依赖

```powershell
go get github.com/google/uuid
```

### 2. 配置文件

复制配置示例文件并修改：

```powershell
Copy-Item config.example.json config.json
```

编辑 `config.json`，配置您的系统密钥和服务器参数。

### 3. 编译运行

```powershell
# 编译
go build -o data-anonymization.exe ./cmd/server

# 运行
.\data-anonymization.exe -config config.json
```

服务将在配置的端口启动（默认8080）。

### 4. 运行测试

```powershell
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/service
go test ./internal/middleware
```

## API使用示例

### 匿名化请求

```bash
POST /v1/anonymize
Authorization: MCP-HMAC-SHA256 SystemID=BI_REPORT_SYSTEM,UserID=user123,Timestamp=1698765432,Signature=...
Content-Type: application/json

{
  "session_id": "sess_12345",
  "payload": {
    "region": "华东",
    "product": "手机",
    "revenue": 1500000
  },
  "anonymization_rules": [
    {
      "strategy": "MAP_CODE",
      "applies_to": {
        "type": "REGION",
        "values": ["华东"]
      }
    }
  ]
}
```

### 解密请求

```bash
POST /v1/decrypt
Authorization: MCP-HMAC-SHA256 SystemID=BI_REPORT_SYSTEM,UserID=user123,Timestamp=1698765432,Signature=...
Content-Type: application/json

{
  "data_with_anonymized_codes": {
    "region": "REGION_a3f5",
    "summary": "REGION_a3f5 表现突出"
  },
  "mappings": {
    "categorical_mappings": {
      "REGION": {
        "REGION_a3f5": "华东"
      }
    }
  }
}
```

## 鉴权说明

所有API请求都需要HMAC签名鉴权：

1. **签名内容**: `SystemID + UserID + UnixTimestamp + SHA256(RequestBody)`
2. **签名算法**: `HMAC-SHA256(SharedSecret, 签名内容)`
3. **Authorization头格式**:
   ```
   MCP-HMAC-SHA256 SystemID={系统ID},UserID={用户ID},Timestamp={时间戳},Signature={签名}
   ```

### 生成签名示例 (Go)

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "strconv"
    "time"
)

func generateSignature(systemID, userID, secret, requestBody string) string {
    // 计算请求体的SHA256
    bodyHash := sha256.Sum256([]byte(requestBody))
    bodyHashStr := hex.EncodeToString(bodyHash[:])
    
    // 生成时间戳
    timestamp := strconv.FormatInt(time.Now().Unix(), 10)
    
    // 构建签名内容
    signContent := systemID + userID + timestamp + bodyHashStr
    
    // 计算HMAC-SHA256
    h := hmac.New(sha256.New, []byte(secret))
    h.Write([]byte(signContent))
    signature := hex.EncodeToString(h.Sum(nil))
    
    return signature
}
```

## 项目结构

```
.
├── cmd/
│   └── server/          # 主程序入口
│       └── main.go
├── internal/
│   ├── config/          # 配置管理
│   ├── handler/         # HTTP处理器
│   ├── logger/          # 日志模块
│   ├── middleware/      # 中间件（鉴权等）
│   └── service/         # 核心业务逻辑
├── docs/                # 文档
├── logs/                # 日志文件目录（自动创建）
├── config.json          # 配置文件
└── README.md
```

## 日志

所有请求都会记录到配置的日志文件中，格式为JSON：

```json
{
  "timestamp": "2025-10-23T08:30:00.123Z",
  "request_id": "uuid-v4",
  "service": "AnonymizationService",
  "system_id": "BI_REPORT_SYSTEM",
  "user_id": "user123",
  "status": "SUCCESS",
  "latency_ms": 15,
  "error_message": null,
  "level": "INFO"
}
```

## 安全建议

1. **密钥管理**: 使用强密钥（至少32字符），定期轮换
2. **HTTPS**: 生产环境必须使用HTTPS
3. **时间戳窗口**: 根据实际需求调整（默认5分钟）
4. **日志保护**: 确保日志文件访问权限受限
5. **配置文件**: 不要将 `config.json` 提交到版本控制系统

## 性能优化

- 匿名化和解密操作都是无状态的，可以水平扩展
- 使用连接池和适当的超时设置
- 考虑使用缓存来存储频繁访问的映射表

## 许可证

MIT License

## 联系方式

如有问题或建议，请提交Issue。