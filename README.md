# Data Anonymization & Decryption Service

[简体中文文档](./docs/README-zh.md) | English

---

A universal microservice for data privacy protection, providing data anonymization and decryption features. Especially suitable for scenarios requiring interaction with third-party AI systems.

## Features

- **Anonymization Service (`/v1/anonymize`)**: Replace sensitive data with secure codes
- **Decryption Service (`/v1/decrypt`)**: Restore anonymized codes to original data
- **HMAC Authentication**: Secure authentication based on HMAC-SHA256
- **Multi-Tenant Support**: Multiple systems can connect independently
- **Structured JSON Logging**
- **Multiple Anonymization Strategies**:
  - MAP_CODE: Mapping code
  - TRANSFORM: Numeric noise addition
  - MAP_PLACEHOLDER: Placeholder mapping
  - PASSTHROUGH: Pass-through

## Tech Stack

- Go 1.24.3
- Gin Web Framework
- Viper (Configuration Management)

## Quick Start

### 1. Install Dependencies

```powershell
go get github.com/google/uuid
```

### 2. Configuration

Copy the example config and edit:

```powershell
Copy-Item config.example.json config.json
```

Edit `config.json` to set your system key and server parameters.

### 3. Build & Run

```powershell
# Build
go build -o data-anonymization.exe ./cmd/server

# Run
.\data-anonymization.exe -config config.json
```

Default port: 8080.

### 4. Run Tests

```powershell
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/service
go test ./internal/middleware
```

## Configuration Parameters

### Server

- **port** (int, required): Listening port (1-65535)
- **log_file** (string, required): Log file path, e.g. `logs/service.log`
- **timestamp_window_seconds** (int, optional): Timestamp validation window (default: 300 seconds)

### Systems

- **system_id** (string, required): Unique system identifier for authentication
- **shared_secret** (string, required): HMAC signing key (at least 32 chars recommended)
- **description** (string, optional): System description

### API Request Parameters

#### Anonymization (`POST /v1/anonymize`)

- **payload** (interface{}, required): Data to anonymize, supports any JSON structure
- **anonymization_rules** (array, required): Array of anonymization rules

**Rule Parameters:**

- **strategy** (string, required): Strategy, options:
  - `MAP_CODE`: Replace string values with random codes
  - `TRANSFORM`: Add random noise to numeric values
  - `MAP_PLACEHOLDER`: Replace numeric values with placeholders
  - `PASSTHROUGH`: Keep original value
- **strategy_params** (map, optional): Strategy-specific parameters
  - For `TRANSFORM`: `noise_level` (float), default 0.05 (5%)
- **applies_to** (object, required): Rule scope
  - `type` (string): Data type, e.g. "REGION", "PRODUCT", "REVENUE"
  - `values` (array): Values to apply the rule to

#### Decryption (`POST /v1/decrypt`)

- **data_with_anonymized_codes** (string, required): Text containing anonymized codes
- **mappings** (map, required): Code-to-original mapping
  - `categorical_mappings` (map): `{type: {code: original}}`
  - `metric_placeholder_mappings` (map): `{placeholder: original}`

## Anonymization Strategies

### 1. MAP_CODE

- For categorical data (e.g. region, product)
- Code format: `{TYPE}_{random 4 hex}` (e.g. `REGION_a3f5`)
- Consistent mapping for same value

### 2. TRANSFORM

- For numeric data, protects exact values
- Algorithm: `new = original + original × noise_level × [-1,1] random`
- Parameter: `noise_level`, default 0.05 (5%)

### 3. MAP_PLACEHOLDER

- For numeric data needing precise restoration
- Placeholder format: `{TYPE}_plc_{index}` (e.g. `USER_COUNT_plc_1`)
- Readable placeholders for AI understanding

### 4. PASSTHROUGH

- Keeps data unchanged
- For non-sensitive data (e.g. growth rate, percentage)

## API Example

### Anonymization Request

```http
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

**Response Example:**

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

### Decryption Request

```http
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

**Response Example:**

```json
{
  "decrypted_data": "分析显示，华东 区域的 手机 表现最佳，活跃用户数为 12000。"
}
```

## Authentication

All API requests require HMAC signature authentication:

1. **Signature Content**: `SystemID + UserID + UnixTimestamp + SHA256(RequestBody)`
2. **Algorithm**: `HMAC-SHA256(SharedSecret, content)`
3. **Authorization Header Format**:
   ```
   MCP-HMAC-SHA256 SystemID={systemID},UserID={userID},Timestamp={timestamp},Signature={signature}
   ```

### Go Example

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "strconv"
    "time"
)

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
```

## Code Format Specification

- **Returned codes do not include braces** (e.g. `REGION_a3f5`)
- **AI must add braces** (e.g. `{REGION_a3f5}`) to mark code positions
- **Decryption only matches codes with braces** for precise replacement
- **Mapping tables store codes without braces** for space efficiency

### Example Flow

1. **Anonymization**: `"华东"` → `"REGION_a3f5"`
2. **AI Processing**: `"REGION_a3f5"` → `"{REGION_a3f5}"`
3. **Decryption**: `"{REGION_a3f5}"` → `"华东"`

## Project Structure

```
.
├── cmd/
│   └── server/          # Main entry
│       └── main.go
├── internal/
│   ├── config/          # Config management
│   ├── handler/         # HTTP handlers
│   ├── logger/          # Logging
│   ├── middleware/      # Middleware (auth etc.)
│   └── service/         # Core logic
├── docs/                # Documentation
├── logs/                # Log files (auto-created)
├── config.json          # Config file
└── README.md
```

## Logging

All requests are logged in JSON format:

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

## Error Handling

Standard HTTP status codes:

- `200`: Success
- `400`: Bad request
- `401`: Unauthorized
- `500`: Internal server error

Error response format:

```json
{
  "error": "Description",
  "code": "ERROR_CODE"
}
```

## Security Recommendations

1. **Key Management**: Use strong keys (≥32 chars), rotate regularly
2. **HTTPS**: Always use HTTPS in production
3. **Timestamp Window**: Adjust as needed (default 5 min)
4. **Log Protection**: Restrict log file access
5. **Config File**: Do not commit `config.json` to version control

## Performance Optimization

- Stateless operations, horizontally scalable
- Use connection pools and proper timeouts
- Consider caching for frequent mapping lookups

## Troubleshooting

- **Auth failure**: Check signature, timestamp, key config
- **Decryption failure**: Check mapping completeness and code format
- **Performance issues**: Use cache for frequent mappings
- **Code mismatch**: Ensure AI adds braces correctly

## Best Practices

1. **Data Type Design**: Design `type` fields for clarity
2. **Strategy Selection**: Choose anonymization strategy by sensitivity
3. **Mapping Management**: Store mappings properly for correct decryption
4. **Test Thoroughly**: Validate anonymization and decryption before integration

## License

MIT License

## Contact

For issues or suggestions, please submit an Issue.

---

[简体中文文档](./docs/README-zh.md)