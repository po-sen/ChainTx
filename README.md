# ChainTx Service

ChainTx 是一個提供「多資產收款請求（Payment Request）」的 HTTP 服務。

目前支援資產：

- `BTC`（bitcoin/regtest, bitcoin/testnet）
- `ETH`（ethereum/sepolia）
- `USDT`（ethereum/sepolia, ERC20）

核心 API：

- `GET /healthz`
- `GET /v1/assets`
- `POST /v1/payment-requests`
- `GET /v1/payment-requests/{id}`

---

## 1. 快速啟動（Docker Compose）

啟動：

```bash
make compose-up
```

停止：

```bash
make compose-down
```

預設服務位址：`http://localhost:8080`

---

## 2. 設定（Environment Variables）

| 變數                                             | 必填             | 預設值             | 說明                                                           |
| ------------------------------------------------ | ---------------- | ------------------ | -------------------------------------------------------------- |
| `DATABASE_URL`                                   | 是               | 無                 | PostgreSQL 連線字串                                            |
| `PORT`                                           | 否               | `8080`             | HTTP 監聽埠                                                    |
| `OPENAPI_SPEC_PATH`                              | 否               | `api/openapi.yaml` | OpenAPI 檔案路徑                                               |
| `PAYMENT_REQUEST_ALLOCATION_MODE`                | 否               | `devtest`          | wallet allocation mode（目前內建 `devtest`、`prod`）           |
| `PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON`           | Devtest 模式必填 | 無                 | keyset JSON（`{"keyset_id":"xpub/tpub/vpub", ...}`）           |
| `PAYMENT_REQUEST_DEVTEST_ALLOW_MAINNET`          | 否               | `false`            | Devtest 是否允許 mainnet allocation                            |
| `PAYMENT_REQUEST_ADDRESS_SCHEME_ALLOW_LIST_JSON` | 否               | 內建 allow-list    | 覆寫 address scheme allow-list（JSON object of string arrays） |

Compose 預設 `DATABASE_URL`：
`postgresql://chaintx:chaintx@postgres:5432/chaintx?sslmode=disable`

---

## 3. 啟動時自動檢查

服務啟動時會依序執行：

1. DB readiness check
2. DB migrations
3. asset catalog integrity validation

任一步驟失敗，服務會直接退出（不對外提供 API）。

---

## 4. API 快速使用

以下命令假設服務在 `http://localhost:8080`。

健康檢查：

```bash
curl -i http://localhost:8080/healthz
```

列出可用資產：

```bash
curl -i http://localhost:8080/v1/assets
```

建立 Payment Request（首次）：

```bash
curl -i \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-key-001' \
  -X POST http://localhost:8080/v1/payment-requests \
  -d '{
    "chain":"bitcoin",
    "network":"testnet",
    "asset":"BTC",
    "expected_amount_minor":"150000",
    "expires_in_seconds":3600,
    "metadata":{"order_id":"A123"}
  }'
```

同 key + 同 payload 重送（idempotency replay）：

- 預期 `200 OK`
- `X-Idempotency-Replayed: true`
- `id` 與首次相同

查詢單筆 Payment Request：

```bash
curl -i http://localhost:8080/v1/payment-requests/<payment_request_id>
```

---

## 5. `POST /v1/payment-requests` 請求重點

主要欄位：

- `chain`, `network`, `asset`（必要）
- `expected_amount_minor`（可選，字串整數，1~78 digits）
- `expires_in_seconds`（可選，範圍 `60..2592000`）
- `metadata`（可選，JSON object，最大 4KB）

建議 Header：

- `Idempotency-Key`：建議必帶
- `X-Principal-ID`：若由 API Gateway 管理租戶/身份，建議注入

---

## 6. 錯誤格式

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

- `400`：格式/驗證錯誤（如 `invalid_request`, `unsupported_asset`, `unsupported_network`）
- `404`：資源不存在（`payment_request_not_found`）
- `409`：idempotency key 與 payload 衝突（`idempotency_key_conflict`）

---

## 7. API 契約文件

- Swagger UI：`GET /swagger/index.html`
- OpenAPI YAML：`GET /swagger/openapi.yaml`

---

## 8. 維運排錯

如果服務啟動即退出，優先檢查：

1. `DATABASE_URL` 是否可連線
2. migration 權限是否足夠
3. asset catalog 與 wallet mapping 是否一致

觀測資訊：

- allocation 結構化 log（含 mode/chain/network/asset/result/latency 等欄位）
