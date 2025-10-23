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

## 详细参数说明

### 配置文件参数

#### Server 配置
- **port** (int, 必需): 服务监听端口，范围 1-65535
- **log_file** (string, 必需): 日志文件路径，如 `logs/service.log`
- **timestamp_window_seconds** (int, 可选): 时间戳验证窗口（秒），默认 300（5分钟）

#### Systems 配置
- **system_id** (string, 必需): 系统唯一标识符，用于鉴权
- **shared_secret** (string, 必需): HMAC签名密钥，建议使用强密钥（至少32字符）
- **description** (string, 可选): 系统描述信息

### API 请求参数

#### 匿名化接口 (`POST /v1/anonymize`)

**请求体参数**:
- **payload** (interface{}, 必需): 需要匿名化的数据，支持任意JSON结构
- **anonymization_rules** (array, 必需): 匿名化规则数组

**匿名化规则参数**:
- **strategy** (string, 必需): 匿名化策略，可选值：
  - `MAP_CODE`: 映射编码策略，将字符串值替换为随机编码
  - `TRANSFORM`: 数值加噪策略，对数值添加随机噪声
  - `MAP_PLACEHOLDER`: 占位符映射策略，将数值替换为占位符
  - `PASSTHROUGH`: 透传策略，保持原始值不变
- **strategy_params** (map, 可选): 策略特定参数
  - 对于 `TRANSFORM` 策略：
    - `noise_level` (float): 噪声级别，默认 0.05（5%）
- **applies_to** (object, 必需): 规则应用范围
  - `type` (string): 数据类型标识符，如 "REGION", "PRODUCT", "REVENUE" 等
  - `values` (array): 需要应用此规则的具体值列表

#### 解密接口 (`POST /v1/decrypt`)

**请求体参数**:
- **data_with_anonymized_codes** (string, 必需): 包含匿名编码的纯文本字符串
- **mappings** (map, 必需): 映射表，包含编码到原始值的映射关系
  - `categorical_mappings` (map): 分类编码映射，格式为 `{类型: {编码: 原始值}}`
  - `metric_placeholder_mappings` (map): 数值占位符映射，格式为 `{占位符: 原始值}`

### 匿名化策略详解

#### 1. MAP_CODE（映射编码策略）
- **用途**: 处理分类数据（如地区、产品名称等）
- **编码格式**: `{TYPE}_{随机4位十六进制}`，如 `REGION_a3f5`
- **特点**: 同一值在不同位置会生成相同编码，确保一致性
- **适用场景**: 地区、产品、用户类型等分类数据

#### 2. TRANSFORM（数值加噪策略）
- **用途**: 处理数值数据，保护精确值
- **算法**: `新值 = 原值 + 原值 × 噪声级别 × [-1,1]随机数`
- **参数**: `noise_level` 控制噪声强度，默认 0.05（5%）
- **适用场景**: 收入、用户数、销售额等数值数据

#### 3. MAP_PLACEHOLDER（占位符映射策略）
- **用途**: 处理需要精确还原的数值数据
- **占位符格式**: `{TYPE}_plc_{序号}`，如 `USER_COUNT_plc_1`
- **特点**: 生成可读的占位符，便于AI理解数据含义
- **适用场景**: 需要精确还原的数值数据

#### 4. PASSTHROUGH（透传策略）
- **用途**: 保持数据不变
- **特点**: 不进行任何处理，直接返回原始值
- **适用场景**: 增长率、百分比等非敏感数据

## API使用示例

### 匿名化请求


**请求示例**:

```bash
POST /v1/anonymize
Authorization: MCP-HMAC-SHA256 SystemID=BI_REPORT_SYSTEM,UserID=user123,Timestamp=1698765432,Signature=...
Content-Type: application/json

{
  "payload": {
    "metadata": {
      "report_name": "Q3 Sales Analysis for {华东}",
      "requester": "user123"
    },
    "analysis_prompt": "Analyze the following sales data. The previous quarter's top product was '手机'. Focus on the performance of '华东' and compare it with other regions. The total revenue for Q2 was 1500000.",
    "data_table": [
      {
        "区域": "华东",
        "核心产品": "手机",
        "季度收入": 1500000,
        "同比增长率": "12.5%",
        "活跃用户数": 12000
      },
      {
        "区域": "华北",
        "核心产品": "电脑",
        "季度收入": 950000,
        "同比增长率": "-3.2%",
        "活跃用户数": 8500
      }
    ]
  },
  "anonymization_rules": [
    {
      "strategy": "MAP_CODE",
      "applies_to": { "type": "REGION", "values": ["华东", "华北"] }
    },
    {
      "strategy": "MAP_CODE",
      "applies_to": { "type": "PRODUCT", "values": ["手机", "电脑"] }
    },
    {
      "strategy": "TRANSFORM",
      "strategy_params": { "noise_level": 0.05 },
      "applies_to": { "type": "REVENUE", "values": [1500000, 950000] }
    },
    {
      "strategy": "MAP_PLACEHOLDER",
      "applies_to": { "type": "USER_COUNT", "values": [12000, 8500] }
    },
    {
      "strategy": "PASSTHROUGH",
      "applies_to": { "type": "GROWTH_RATE", "values": ["12.5%", "-3.2%"] }
    }
  ]
}
```

**响应示例**:

```json
{
  "anonymized_payload": {
    "metadata": {
      "report_name": "Q3 Sales Analysis for {REGION_a3f5}",
      "requester": "user123"
    },
    "analysis_prompt": "Analyze the following sales data. The previous quarter's top product was 'PRODUCT_c8b1'. Focus on the performance of 'REGION_a3f5' and compare it with other regions. The total revenue for Q2 was 1532108.5.",
    "data_table": [
      {
        "区域": "REGION_a3f5",
        "核心产品": "PRODUCT_c8b1",
        "季度收入": 1532108.5,
        "同比增长率": "12.5%",
        "活跃用户数": "USER_COUNT_plc_1"
      },
      {
        "区域": "REGION_b1e9",
        "核心产品": "PRODUCT_d2a7",
        "季度收入": 988450.0,
        "同比增长率": "-3.2%",
        "活跃用户数": "USER_COUNT_plc_2"
      }
    ]
  },
  "mappings_to_store": {
    "categorical_mappings": {
      "REGION": { "REGION_a3f5": "华东", "REGION_b1e9": "华北" },
      "PRODUCT": { "PRODUCT_c8b1": "手机", "PRODUCT_d2a7": "电脑" }
    },
    "metric_placeholder_mappings": {
      "USER_COUNT_plc_1": 12000,
      "USER_COUNT_plc_2": 8500
    }
  }
}
```

### 解密请求

**请求示例**:

```bash
POST /v1/decrypt
Authorization: MCP-HMAC-SHA256 SystemID=BI_REPORT_SYSTEM,UserID=user123,Timestamp=1698765432,Signature=...
Content-Type: application/json

{
  "data_with_anonymized_codes": "分析显示，{REGION_a3f5} 区域的 {PRODUCT_c8b1} 表现最佳，活跃用户数为 {USER_COUNT_plc_1}。",
  "mappings": {
    "categorical_mappings": {
      "REGION": { "REGION_a3f5": "华东", "REGION_b1e9": "华北" },
      "PRODUCT": { "PRODUCT_c8b1": "手机", "PRODUCT_d2a7": "电脑" }
    },
    "metric_placeholder_mappings": {
      "USER_COUNT_plc_1": 12000,
      "USER_COUNT_plc_2": 8500
    }
  }
}
```

**响应示例**:

```json
{
  "decrypted_data": "分析显示，华东 区域的 手机 表现最佳，活跃用户数为 12000。"
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

## 编码格式规范

### 编码生成规则
- **匿名化服务返回的编码不带大括号**（如 `REGION_a3f5`）
- **AI在处理时必须主动添加大括号**（如 `{REGION_a3f5}`）来标记编码位置
- **解密时只匹配带大括号的编码**，确保精确替换
- **映射表存储不带大括号的编码**，节省存储空间

### 示例流程
1. **匿名化**: `"华东"` → `"REGION_a3f5"`
2. **AI处理**: `"REGION_a3f5"` → `"{REGION_a3f5}"`
3. **解密**: `"{REGION_a3f5}"` → `"华东"`

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

## 错误处理

服务返回标准HTTP状态码：

- `200`: 成功
- `400`: 请求参数错误
- `401`: 鉴权失败
- `500`: 服务器内部错误

错误响应格式：

```json
{
  "error": "错误描述",
  "code": "ERROR_CODE"
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

## 故障排除

- **鉴权失败**: 检查签名算法、时间戳和密钥配置
- **解密失败**: 确认映射表完整性和编码格式
- **性能问题**: 考虑使用缓存存储频繁访问的映射表
- **编码不匹配**: 确保AI在处理时正确添加大括号

## 最佳实践

1. **数据类型设计**: 合理设计 `type` 字段，便于管理和理解
2. **策略选择**: 根据数据敏感性选择合适的匿名化策略
3. **映射表管理**: 妥善存储和管理映射表，确保解密时能够正确还原数据
4. **测试验证**: 在集成前充分测试匿名化和解密流程

## 许可证

MIT License

## 联系方式

如有问题或建议，请提交Issue。