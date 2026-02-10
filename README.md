# ChainTx Service

ChainTx 是一個提供「多資產收款請求（Payment Request）」的 HTTP 服務。
目前支援：

- `BTC`（bitcoin/mainnet）
- `ETH`（ethereum/mainnet）
- `USDT`（ethereum/mainnet, ERC20）

核心 API：

- `GET /healthz`
- `GET /v1/assets`
- `POST /v1/payment-requests`
- `GET /v1/payment-requests/{id}`

---

## 1. 部署前需求

- PostgreSQL（建議 14+）
- 可用的 `DATABASE_URL`
- 若使用容器：Docker Engine + Docker Compose plugin
- 若使用原生執行：Go `1.25.7`

---

## 2. 設定（Environment Variables）

| 變數                | 必填 | 預設值             | 說明                |
| ------------------- | ---- | ------------------ | ------------------- |
| `DATABASE_URL`      | 是   | 無                 | PostgreSQL 連線字串 |
| `PORT`              | 否   | `8080`             | HTTP 監聽埠         |
| `OPENAPI_SPEC_PATH` | 否   | `api/openapi.yaml` | OpenAPI 檔案路徑    |

Compose 預設 `DATABASE_URL`：
`postgresql://chaintx:chaintx@postgres:5432/chaintx?sslmode=disable`

---

## 3. 服務啟動流程（重要）

服務啟動時會依序執行：

1. DB readiness check（可連線）
2. DB migrations（`golang-migrate`）
3. asset catalog integrity validation（資產目錄與 wallet mapping 一致性）

任一步驟失敗會直接退出（non-zero exit），不會開始對外服務。

---

## 4. 部署方式

### A. Docker Compose（最快）

```bash
make compose-up
```

停止：

```bash
make compose-down
```

### B. 容器映像（自建）

```bash
docker build -f build/package/Dockerfile -t chaintx:latest .

docker run --rm \
  -p 8080:8080 \
  -e DATABASE_URL='postgresql://chaintx:chaintx@<db-host>:5432/chaintx?sslmode=disable' \
  -e PORT=8080 \
  -e OPENAPI_SPEC_PATH=/app/api/openapi.yaml \
  chaintx:latest
```

### C. 原生執行（Binary）

```bash
go build -o bin/server ./cmd/server

DATABASE_URL='postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable' \
PORT=8080 \
OPENAPI_SPEC_PATH=api/openapi.yaml \
./bin/server
```

---

## 5. 部署後 Smoke Test

> 以下命令假設服務在 `http://localhost:8080`

### 5.1 健康檢查

```bash
curl -i http://localhost:8080/healthz
```

預期：`200 OK`，body 含 `{"status":"ok"}`

### 5.2 讀取可用資產

```bash
curl -i http://localhost:8080/v1/assets
```

預期：`200 OK`，回傳 `assets` 陣列（含 BTC/ETH/USDT）

### 5.3 建立 Payment Request（首次）

```bash
curl -i \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-key-001' \
  -X POST http://localhost:8080/v1/payment-requests \
  -d '{
    "chain":"bitcoin",
    "network":"mainnet",
    "asset":"BTC",
    "expected_amount_minor":"150000",
    "expires_in_seconds":3600,
    "metadata":{"order_id":"A123"}
  }'
```

預期：

- `201 Created`
- `Location: /v1/payment-requests/{id}`
- response 含 `payment_instructions`

### 5.4 同 key + 同 payload 重送（idempotency replay）

重送完全相同請求，預期：

- `200 OK`
- `X-Idempotency-Replayed: true`
- `Location` 與 `id` 與首次一致

### 5.5 讀取指定 Payment Request

```bash
curl -i http://localhost:8080/v1/payment-requests/<payment_request_id>
```

預期：`200 OK`，回傳該筆完整資源；不存在則 `404`

---

## 6. API 使用重點（給整合方）

### `GET /v1/assets`

提供 server side catalog，不應由客戶端硬編碼 decimals / token metadata。

### `POST /v1/payment-requests`

Request 重要欄位：

- `chain`, `network`, `asset`（必要）
- `expected_amount_minor`（可選，字串整數，1~78 digits）
- `expires_in_seconds`（可選，範圍 `60..2592000`）
- `metadata`（可選，JSON object，最大 4KB）

建議 header：

- `Idempotency-Key`：強烈建議傳
- `X-Principal-ID`：若由 API Gateway 管理身份，建議注入此值，避免不同租戶 key scope 混用

### `GET /v1/payment-requests/{id}`

用於查詢已建立的 payment request 狀態與指示資訊。

---

## 7. 錯誤格式

所有 API 錯誤統一格式：

```json
{
  "error": {
    "code": "invalid_request",
    "message": "expires_in_seconds must be between 60 and 2592000",
    "details": {
      "field": "expires_in_seconds"
    }
  }
}
```

常見狀態碼：

- `400`：格式/驗證錯誤（`invalid_request`, `unsupported_asset`, `unsupported_network`）
- `404`：資源不存在（`payment_request_not_found`）
- `409`：idempotency key 與 payload 衝突（`idempotency_key_conflict`）

---

## 8. OpenAPI / Swagger

- Swagger UI：`GET /swagger/index.html`
- OpenAPI YAML：`GET /swagger/openapi.yaml`
- 原始契約檔案：`api/openapi.yaml`

---

## 9. 維運與排錯

- 啟動即退出：
  - 先檢查 `DATABASE_URL` 是否可連線
  - 檢查 migration 權限
  - 檢查 asset catalog / wallet mapping 是否一致（啟動驗證會擋下錯誤配置）
- 本地資料重置（僅開發環境）：

```bash
docker compose -f deployments/docker-compose.yml down -v
```

---

## 10. 主要程式結構

```text
cmd/server/main.go                               # entrypoint orchestration
internal/domain                                  # domain objects + policies
internal/application                             # use cases + ports + DTOs
internal/adapters/inbound                        # HTTP controllers/router
internal/adapters/outbound/persistence/postgresql/assetcatalog
                                                 # asset catalog read-model adapter
internal/adapters/outbound/persistence/postgresql/paymentrequest
                                                 # payment request repository/read-model adapter
internal/adapters/outbound/persistence/postgresql/bootstrap
                                                 # startup readiness/migration/integrity gateway
internal/adapters/outbound/persistence/postgresql/shared
                                                 # shared db pool utilities
internal/adapters/outbound/wallet/deterministic # address allocator adapter
internal/bootstrap                               # config + DI composition root
api/openapi.yaml                                 # OpenAPI contract
deployments/docker-compose.yml                   # compose deployment
build/package/Dockerfile                         # image build
```
