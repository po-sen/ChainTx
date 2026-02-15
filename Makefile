SHELL := /bin/bash

DOCKER_COMPOSE ?= docker compose

SERVICE_COMPOSE := deployments/service/docker-compose.yml
BTC_COMPOSE := deployments/local-chains/docker-compose.btc.yml
ETH_COMPOSE := deployments/local-chains/docker-compose.eth.yml

SERVICE_PROJECT := chaintx-local-service
BTC_PROJECT := chaintx-local-btc
ETH_PROJECT := chaintx-local-eth

ARTIFACT_DIR := deployments/local-chains/artifacts
SCRIPT_DIR := scripts/local-chains
BTC_ARTIFACT := $(ARTIFACT_DIR)/btc.json
ETH_ARTIFACT := $(ARTIFACT_DIR)/eth.json
SMOKE_LOCAL_ARTIFACT := $(ARTIFACT_DIR)/smoke-local.json
SMOKE_ALL_ARTIFACT := $(ARTIFACT_DIR)/smoke-local-all.json

BTC_RPC_USER ?= chaintx
BTC_RPC_PASSWORD ?= chaintx
BTC_RPC_URL ?= http://127.0.0.1:18443
ETH_RPC_URL ?= http://127.0.0.1:8545

SERVICE_KS_BTC_REGTEST ?= tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5
SERVICE_KS_BTC_TESTNET ?= vpub5Xzfrm6ouSBPKVriRpkXyai4mvsHjRHq28wxS1znBCdwzLzeJUx8ruJeBnCMKs1AyqYsJ2mriQHuzxNoFtkkq94J4bJyNjGXkbZ8vCYwUy3
SERVICE_KS_ETH_SEPOLIA ?= xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj
SERVICE_KS_ETH_LOCAL ?= $(SERVICE_KS_ETH_SEPOLIA)

SERVICE_DC := $(DOCKER_COMPOSE) -f $(SERVICE_COMPOSE) --project-name $(SERVICE_PROJECT)
BTC_DC := $(DOCKER_COMPOSE) -f $(BTC_COMPOSE) --project-name $(BTC_PROJECT)
ETH_DC := $(DOCKER_COMPOSE) -f $(ETH_COMPOSE) --project-name $(ETH_PROJECT)

.PHONY: \
	service-up service-down \
	chain-up-btc chain-down-btc \
	chain-up-eth chain-down-eth \
	chain-up-all chain-down-all \
	local-up local-up-all local-down

service-up:
	@set -eu; \
	keysets_json="$$(BTC_ARTIFACT_FILE="$(BTC_ARTIFACT)" ETH_ARTIFACT_FILE="$(ETH_ARTIFACT)" SERVICE_KS_BTC_REGTEST="$(SERVICE_KS_BTC_REGTEST)" SERVICE_KS_BTC_TESTNET="$(SERVICE_KS_BTC_TESTNET)" SERVICE_KS_ETH_SEPOLIA="$(SERVICE_KS_ETH_SEPOLIA)" SERVICE_KS_ETH_LOCAL="$(SERVICE_KS_ETH_LOCAL)" $(SCRIPT_DIR)/service_keysets_json.sh)"; \
	PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON="$$keysets_json" $(SERVICE_DC) up -d --build postgres app; \
	SERVICE_COMPOSE_FILE="$(SERVICE_COMPOSE)" LOCAL_SERVICE_PROJECT="$(SERVICE_PROJECT)" ETH_ARTIFACT_FILE="$(ETH_ARTIFACT)" $(SCRIPT_DIR)/service_sync_catalog.sh

service-down:
	$(SERVICE_DC) stop app postgres

chain-up-btc:
	$(BTC_DC) up -d btc-node
	LOCAL_CHAIN_ARTIFACT_DIR="$(ARTIFACT_DIR)" LOCAL_BTC_PROJECT="$(BTC_PROJECT)" BTC_COMPOSE_FILE="$(BTC_COMPOSE)" BTC_RPC_USER="$(BTC_RPC_USER)" BTC_RPC_PASSWORD="$(BTC_RPC_PASSWORD)" BTC_RPC_URL="$(BTC_RPC_URL)" $(SCRIPT_DIR)/btc_bootstrap.sh

chain-down-btc:
	$(BTC_DC) stop btc-node
	rm -f $(BTC_ARTIFACT) $(SMOKE_LOCAL_ARTIFACT) $(SMOKE_ALL_ARTIFACT)

chain-up-eth:
	$(ETH_DC) up -d eth-node
	LOCAL_CHAIN_ARTIFACT_DIR="$(ARTIFACT_DIR)" LOCAL_ETH_PROJECT="$(ETH_PROJECT)" ETH_COMPOSE_FILE="$(ETH_COMPOSE)" ETH_RPC_URL="$(ETH_RPC_URL)" ETH_EXPECTED_CHAIN_ID="31337" SERVICE_KS_ETH_SEPOLIA="$(SERVICE_KS_ETH_SEPOLIA)" $(SCRIPT_DIR)/eth_export_artifacts.sh
	ETH_RPC_URL="http://eth-node:8545" ETH_EXPECTED_CHAIN_ID="31337" $(ETH_DC) run --rm usdt-deployer

chain-down-eth:
	$(ETH_DC) stop eth-node
	rm -f $(ETH_ARTIFACT) $(ARTIFACT_DIR)/usdt.json $(SMOKE_ALL_ARTIFACT)

chain-up-all: chain-up-btc chain-up-eth

chain-down-all: chain-down-eth chain-down-btc

local-up: chain-up-btc service-up

local-up-all: chain-up-all service-up

local-down: service-down chain-down-all
