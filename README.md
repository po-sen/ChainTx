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

命名規則（build/deployments）：

- `build/service/*`：service image build files
- `deployments/service/*`：service stack compose
- `deployments/local-chains/*`：local chain compose（btc/eth）

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
- `btc-explorer`（BTC 區塊瀏覽器）

`local-up`/`local-up-all` 在啟動 `service` 時，會優先讀取 artifacts 並注入：

- `deployments/local-chains/artifacts/btc.json` `receiver_xpub` -> `ks_btc_regtest`
- `deployments/local-chains/artifacts/eth.json` `receiver_xpub` -> `ks_eth_local`
- `ks_eth_sepolia` 使用固定 devtest keyset（不跟 local artifact 混用）

此外，`service-up` 會同步 DB catalog：

- 重建/修正 `ethereum/sepolia` baseline（固定 `chain_id=11155111`）
- upsert `wallet_accounts(wa_eth_local_001 -> ks_eth_local)`
- upsert `asset_catalog(ethereum/local/ETH)`（`chain_id=31337`）
- upsert `asset_catalog(ethereum/local/USDT)`（`token_contract`/`token_decimals` 來自 `eth.json` 的 USDT 欄位）

`ethereum/sepolia` rows 仍保留，不會被 local sync 覆蓋。

全量 profile（需要 ETH + USDT 測試時）：

```bash
make local-up-all
```

會額外啟動：

- `eth` stack（anvil, chain id 31337）
- `eth-explorer`（EVM 區塊瀏覽器）
- `usdt` deploy step（`chain-up-eth` 內建一次性 `usdt-deployer`，把 USDT 合約部署到同一條 `eth-node` 鏈上）

停止 profile：

```bash
make local-down
```

### Explorer Deployment Rules

可用不同 Make target 決定是否部署 explorer：

- `make chain-up-btc`：用 `COMPOSE_PROFILES=btc-explorer` 啟動 BTC node + BTC explorer
- `make chain-up-btc-no-explorer`：只啟動 BTC node
- `make chain-up-eth`：用 `COMPOSE_PROFILES=eth-explorer` 啟動 ETH node + Blockscout（db/redis/backend/frontend）
- `make chain-up-eth-no-explorer`：只啟動 ETH node（仍會執行 USDT deploy）
- `make chain-up-all`：BTC/ETH 都含 explorer
- `make chain-up-all-no-explorer`：BTC/ETH 都不含 explorer
- `make local-up` / `make local-up-all`：含 explorer
- `make local-up-no-explorer` / `make local-up-all-no-explorer`：不含 explorer

如果你直接使用 `docker compose`，也可以用 profiles：

- BTC explorer profile: `btc-explorer`（也支援 `explorer`）
- ETH explorer profile: `eth-explorer`（也支援 `explorer`）
- USDT deployer profile: `usdt-deployer`（one-shot；Makefile 會用 `run --rm usdt-deployer` 觸發）

範例：

```bash
# BTC: 只起 node
docker compose -f deployments/local-chains/docker-compose.btc.yml up -d btc-node

# BTC: 起 node + explorer
docker compose -f deployments/local-chains/docker-compose.btc.yml --profile btc-explorer up -d

# ETH: 只起 anvil node
docker compose -f deployments/local-chains/docker-compose.eth.yml up -d eth-node

# ETH: 起 anvil + blockscout stack
docker compose -f deployments/local-chains/docker-compose.eth.yml --profile eth-explorer up -d
```

### Per-rail Commands

BTC:

```bash
make chain-up-btc
make chain-up-btc-no-explorer
make chain-down-btc
```

ETH:

```bash
make chain-up-eth
make chain-up-eth-no-explorer
make chain-down-eth
```

Service:

```bash
make service-up
make service-down
```

Aggregate:

```bash
make chain-up-all
make chain-up-all-no-explorer
make chain-down-all
```

### Local Explorer URLs

- BTC explorer: `http://127.0.0.1:${BTC_EXPLORER_PORT:-3002}`
- ETH explorer (Blockscout): `http://127.0.0.1:${ETH_EXPLORER_PORT:-5100}`

可覆寫的埠位與 RPC 參數：

- `BTC_EXPLORER_PORT`（預設 `3002`）
- `ETH_EXPLORER_PORT`（預設 `5100`，Blockscout frontend）
- `ETH_EXPLORER_API_PORT`（預設 `5101`，Blockscout backend API）

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

## Local Manual Receive Test Runbook

以下流程可完整驗證「服務產生收款地址」與「鏈上實際收到款」。

### 1. 啟動所有必要服務

```bash
make local-up-all
```

確認狀態：

```bash
docker ps --format "{{.Names}}\t{{.Status}}" | rg '^chaintx-local-'
curl -sS http://127.0.0.1:8080/healthz
curl -sS http://127.0.0.1:8080/v1/assets | jq
```

### 2. BTC（regtest）手動收款測試

建立 BTC payment request：

```bash
PR=$(curl -sS -X POST http://127.0.0.1:8080/v1/payment-requests \
  -H 'Content-Type: application/json' \
  -d '{"chain":"bitcoin","network":"regtest","asset":"BTC","expected_amount_minor":"50000"}')

PR_ID=$(echo "$PR" | jq -r '.id')
BTC_ADDR=$(echo "$PR" | jq -r '.payment_instructions.address')
echo "$PR" | jq '{id,status,address:.payment_instructions.address}'
```

從 payer wallet 轉帳到收款地址：

```bash
TXID=$(docker compose -f deployments/local-chains/docker-compose.btc.yml \
  --project-name chaintx-local-btc \
  exec -T btc-node bitcoin-cli -regtest -rpcuser=chaintx -rpcpassword=chaintx \
  -rpcwallet=chaintx-btc-payer sendtoaddress "$BTC_ADDR" 0.001)

echo "$TXID"
```

挖 1 個區塊確認交易：

```bash
MINE_ADDR=$(docker compose -f deployments/local-chains/docker-compose.btc.yml \
  --project-name chaintx-local-btc \
  exec -T btc-node bitcoin-cli -regtest -rpcuser=chaintx -rpcpassword=chaintx \
  -rpcwallet=chaintx-btc-payer getnewaddress "" bech32 | tr -d '\r')

docker compose -f deployments/local-chains/docker-compose.btc.yml \
  --project-name chaintx-local-btc \
  exec -T btc-node bitcoin-cli -regtest -rpcuser=chaintx -rpcpassword=chaintx \
  -rpcwallet=chaintx-btc-payer generatetoaddress 1 "$MINE_ADDR" >/dev/null

docker compose -f deployments/local-chains/docker-compose.btc.yml \
  --project-name chaintx-local-btc \
  exec -T btc-node bitcoin-cli -regtest -rpcuser=chaintx -rpcpassword=chaintx \
  -rpcwallet=chaintx-btc-payer gettransaction "$TXID" | jq '{txid,confirmations,details}'
```

回查 payment request：

```bash
curl -sS "http://127.0.0.1:8080/v1/payment-requests/$PR_ID" | jq
```

### 3. ETH（local EVM）手動收款測試

建立 ETH payment request：

```bash
PR=$(curl -sS -X POST http://127.0.0.1:8080/v1/payment-requests \
  -H 'Content-Type: application/json' \
  -d '{"chain":"ethereum","network":"local","asset":"ETH","expected_amount_minor":"1000000000000000"}')

ETH_ADDR=$(echo "$PR" | jq -r '.payment_instructions.address')
echo "$PR" | jq '{id,status,address:.payment_instructions.address}'
```

轉 0.001 ETH 到收款地址：

```bash
docker compose -f deployments/local-chains/docker-compose.eth.yml \
  --project-name chaintx-local-eth \
  exec -T eth-node cast send \
  --rpc-url http://127.0.0.1:8545 \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
  "$ETH_ADDR" --value 1000000000000000 --json | jq -r '.transactionHash'
```

檢查收款地址餘額：

```bash
docker compose -f deployments/local-chains/docker-compose.eth.yml \
  --project-name chaintx-local-eth \
  exec -T eth-node cast balance --rpc-url http://127.0.0.1:8545 "$ETH_ADDR"
```

### 4. USDT（ERC20）手動收款測試

建立 USDT payment request：

```bash
PR=$(curl -sS -X POST http://127.0.0.1:8080/v1/payment-requests \
  -H 'Content-Type: application/json' \
  -d '{"chain":"ethereum","network":"local","asset":"USDT","expected_amount_minor":"1000000"}')

USDT_ADDR=$(echo "$PR" | jq -r '.payment_instructions.address')
echo "$PR" | jq '{id,status,address:.payment_instructions.address}'
```

讀取 USDT 合約地址並轉 1 USDT（`1000000`，decimals=6）：

```bash
USDT_CONTRACT=$(jq -r '.usdt_contract_address' deployments/local-chains/artifacts/eth.json)

docker compose -f deployments/local-chains/docker-compose.eth.yml \
  --project-name chaintx-local-eth \
  exec -T eth-node cast send \
  --rpc-url http://127.0.0.1:8545 \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
  "$USDT_CONTRACT" "transfer(address,uint256)" "$USDT_ADDR" 1000000 --json | jq -r '.transactionHash'
```

檢查收款地址 USDT 餘額：

```bash
docker compose -f deployments/local-chains/docker-compose.eth.yml \
  --project-name chaintx-local-eth \
  exec -T eth-node cast call \
  --rpc-url http://127.0.0.1:8545 \
  "$USDT_CONTRACT" "balanceOf(address)(uint256)" "$USDT_ADDR"
```

### 5. 一鍵 smoke（可選）

```bash
scripts/local-chains/smoke_local_all.sh
```

輸出檔案：

- `deployments/local-chains/artifacts/smoke-local.json`
- `deployments/local-chains/artifacts/smoke-local-all.json`

### 6. 關閉

```bash
make local-down
```

## Troubleshooting

- `chain-up-eth` 失敗且訊息為 chain mismatch：確認 `eth-node` chain id 為 `31337`，再重跑 `make chain-down-eth && make chain-up-eth`。
- full smoke 顯示 USDT stale artifact：執行 `make chain-down-eth && make chain-up-eth`（會重建 ETH+USDT artifacts）。
- `service-up` 顯示 `invalid eth artifact usdt_*`：先重跑 `make chain-down-eth && make chain-up-eth`，再執行 `make service-up`。
- BTC 餘額不足：重跑 `make chain-up-btc`（bootstrap 會自動補挖）。
- 服務啟動失敗：用 `docker compose -f deployments/service/docker-compose.yml --project-name chaintx-local-service logs app postgres` 檢查詳細訊息。
