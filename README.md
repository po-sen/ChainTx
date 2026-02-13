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

## Prerequisites

- Go `1.25.7+`
- Docker Engine + Docker Compose plugin
- `jq`
- `curl`

## Service-only Workflow

只啟動 service stack（app + postgres）：

```bash
make service-up
```

停止：

```bash
make service-down
```

## Local Chain Simulation (new workflow)

此模式提供獨立 rail stacks，遵守以下固定常數：

- BTC local: `regtest` only
- EVM local: `chain_id=31337`
- USDT local token decimals: `6`
- USDT local EVM RPC: `USDT_RPC_URL`（預設 `http://127.0.0.1:8546`）

命名規則（build/deployments）：

- `build/service/*`：service image build files
- `build/local-chains/*`：local chain helper image build files
- `deployments/service/*`：service stack compose
- `deployments/local-chains/*`：chain rail compose（btc/eth/usdt）

擴充規則（保持低耦合）：

- 新增 rail（例如 `usdt-tron`）時，新增 `docker-compose.<rail>.yml`、`chain-up/down-<rail>`、`<rail>.json` artifact 即可。
- 既有 rail（btc/eth/usdt）不需要修改；最多只調整 `chain-up-all` / `chain-down-all` 聚合目標。

### Profiles

預設最小 profile（建議日常開發）：

```bash
make local-up
```

會啟動：

- `service` stack（app + postgres）
- `btc` stack（含 payer/receiver descriptor bootstrap）

全量 profile（需要 ETH + USDT 測試時）：

```bash
make local-up-all
```

會額外啟動：

- `eth` stack（anvil, chain id 31337）
- `usdt` stack（單一 `usdt-node` 容器內含 anvil + deploy/mint, decimals=6）

停止 profile：

```bash
make local-down
```

### Per-rail Commands

BTC:

```bash
make chain-up-btc
make chain-down-btc
```

ETH:

```bash
make chain-up-eth
make chain-down-eth
```

USDT:

```bash
make chain-up-usdt
make chain-down-usdt
```

`chain-up-usdt` 會啟動單一 `usdt-node`（內含 USDT 專用 EVM，預設 host `:8546`），等待 deploy/healthcheck 完成後常駐執行。

Service:

```bash
make service-up
make service-down
```

Aggregate:

```bash
make chain-up-all
make chain-down-all
```

## Configuration

| Variable                                         | Required     | Default            | Description                                         |
| ------------------------------------------------ | ------------ | ------------------ | --------------------------------------------------- |
| `DATABASE_URL`                                   | Yes          | none               | PostgreSQL DSN                                      |
| `PORT`                                           | No           | `8080`             | HTTP listen port                                    |
| `OPENAPI_SPEC_PATH`                              | No           | `api/openapi.yaml` | OpenAPI file path                                   |
| `PAYMENT_REQUEST_ALLOCATION_MODE`                | No           | `devtest`          | Wallet allocation mode (`devtest`, `prod`)          |
| `PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON`           | Devtest only | none               | Keyset JSON (`{"keyset_id":"xpub/tpub/vpub", ...}`) |
| `PAYMENT_REQUEST_DEVTEST_ALLOW_MAINNET`          | No           | `false`            | Allow mainnet allocation in devtest mode            |
| `PAYMENT_REQUEST_ADDRESS_SCHEME_ALLOW_LIST_JSON` | No           | built-in allowlist | Override address-scheme allowlist                   |

## API Quick Usage

以下命令假設服務位於 `http://localhost:8080`。

健康檢查：

```bash
curl -i http://localhost:8080/healthz
```

列出資產：

```bash
curl -i http://localhost:8080/v1/assets
```

建立 Payment Request：

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

## Troubleshooting

- `chain-up-usdt` 失敗且訊息為 chain mismatch：重跑 `make chain-down-usdt && make chain-up-usdt`（USDT rail 與 ETH rail 已分離，不需依賴 `chain-up-eth`）。
- `chain-up-usdt` 或 full smoke 顯示 USDT stale artifact：執行 `make chain-down-usdt && make chain-up-usdt`。
- BTC 餘額不足：重跑 `make chain-up-btc`（bootstrap 會自動補挖）。
- 服務啟動失敗：用 `docker compose -f deployments/service/docker-compose.yml --project-name chaintx-local-service logs app postgres` 檢查詳細訊息。
