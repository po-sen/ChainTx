SHELL := /bin/bash

DOCKER_COMPOSE ?= docker compose

LOCAL_CHAIN_ARTIFACT_DIR ?= deployments/local-chains/artifacts
LOCAL_CHAIN_SCRIPT_DIR ?= scripts/local-chains
BTC_ARTIFACT_FILE ?= $(LOCAL_CHAIN_ARTIFACT_DIR)/btc.json
ETH_ARTIFACT_FILE ?= $(LOCAL_CHAIN_ARTIFACT_DIR)/eth.json
USDT_ARTIFACT_FILE ?= $(LOCAL_CHAIN_ARTIFACT_DIR)/usdt.json
SMOKE_LOCAL_ARTIFACT_FILE ?= $(LOCAL_CHAIN_ARTIFACT_DIR)/smoke-local.json
SMOKE_LOCAL_ALL_ARTIFACT_FILE ?= $(LOCAL_CHAIN_ARTIFACT_DIR)/smoke-local-all.json

LOCAL_SERVICE_PROJECT ?= chaintx-local-service
LOCAL_BTC_PROJECT ?= chaintx-local-btc
LOCAL_ETH_PROJECT ?= chaintx-local-eth
LOCAL_USDT_PROJECT ?= chaintx-local-usdt

LOCAL_SERVICE_COMPOSE_FILE ?= deployments/service/docker-compose.yml
LOCAL_BTC_COMPOSE_FILE ?= deployments/local-chains/docker-compose.btc.yml
LOCAL_ETH_COMPOSE_FILE ?= deployments/local-chains/docker-compose.eth.yml
LOCAL_USDT_COMPOSE_FILE ?= deployments/local-chains/docker-compose.usdt.yml

BTC_RPC_PORT ?= 18443
ETH_RPC_PORT ?= 8545
USDT_RPC_PORT ?= 8546
ETH_CHAIN_ID ?= 31337
USDT_CHAIN_ID ?= 31337
BTC_RPC_USER ?= chaintx
BTC_RPC_PASSWORD ?= chaintx

BTC_RPC_URL ?= http://127.0.0.1:$(BTC_RPC_PORT)
ETH_RPC_URL ?= http://127.0.0.1:$(ETH_RPC_PORT)
USDT_RPC_URL ?= http://127.0.0.1:$(USDT_RPC_PORT)

SERVICE_DC = $(DOCKER_COMPOSE) -f $(LOCAL_SERVICE_COMPOSE_FILE) --project-name $(LOCAL_SERVICE_PROJECT)
BTC_DC = $(DOCKER_COMPOSE) -f $(LOCAL_BTC_COMPOSE_FILE) --project-name $(LOCAL_BTC_PROJECT)
ETH_DC = $(DOCKER_COMPOSE) -f $(LOCAL_ETH_COMPOSE_FILE) --project-name $(LOCAL_ETH_PROJECT)
USDT_DC = $(DOCKER_COMPOSE) -f $(LOCAL_USDT_COMPOSE_FILE) --project-name $(LOCAL_USDT_PROJECT)

CHAIN_SCRIPT_ENV = \
	LOCAL_CHAIN_ARTIFACT_DIR=$(LOCAL_CHAIN_ARTIFACT_DIR) \
	LOCAL_BTC_PROJECT=$(LOCAL_BTC_PROJECT) \
	LOCAL_ETH_PROJECT=$(LOCAL_ETH_PROJECT) \
	LOCAL_USDT_PROJECT=$(LOCAL_USDT_PROJECT) \
	BTC_COMPOSE_FILE=$(LOCAL_BTC_COMPOSE_FILE) \
	ETH_COMPOSE_FILE=$(LOCAL_ETH_COMPOSE_FILE) \
	USDT_COMPOSE_FILE=$(LOCAL_USDT_COMPOSE_FILE) \
	BTC_RPC_USER=$(BTC_RPC_USER) \
	BTC_RPC_PASSWORD=$(BTC_RPC_PASSWORD) \
	BTC_RPC_URL=$(BTC_RPC_URL) \
	ETH_RPC_URL=$(ETH_RPC_URL) \
	ETH_EXPECTED_CHAIN_ID=$(ETH_CHAIN_ID)

.PHONY: \
	service-up service-down \
	chain-up-btc chain-down-btc \
	chain-up-eth chain-down-eth \
	chain-up-usdt chain-down-usdt \
	chain-up-all chain-down-all \
	local-up local-up-all local-down

service-up:
	$(SERVICE_DC) up -d postgres app

service-down:
	$(SERVICE_DC) stop app postgres

chain-up-btc:
	$(BTC_DC) up -d btc-node
	$(CHAIN_SCRIPT_ENV) $(LOCAL_CHAIN_SCRIPT_DIR)/btc_bootstrap.sh

chain-down-btc:
	$(BTC_DC) stop btc-node
	rm -f $(BTC_ARTIFACT_FILE) $(SMOKE_LOCAL_ARTIFACT_FILE) $(SMOKE_LOCAL_ALL_ARTIFACT_FILE)

chain-up-eth:
	$(ETH_DC) up -d eth-node
	$(CHAIN_SCRIPT_ENV) $(LOCAL_CHAIN_SCRIPT_DIR)/eth_export_artifacts.sh

chain-down-eth:
	$(ETH_DC) stop eth-node
	rm -f $(ETH_ARTIFACT_FILE) $(SMOKE_LOCAL_ALL_ARTIFACT_FILE)

chain-up-usdt:
	USDT_RPC_PORT=$(USDT_RPC_PORT) USDT_CHAIN_ID=$(USDT_CHAIN_ID) USDT_ARTIFACT_RPC_URL=$(USDT_RPC_URL) $(USDT_DC) up --build -d --wait --wait-timeout 180 usdt-node

chain-down-usdt:
	$(USDT_DC) down --remove-orphans
	rm -f $(USDT_ARTIFACT_FILE) $(SMOKE_LOCAL_ALL_ARTIFACT_FILE)

chain-up-all: chain-up-btc chain-up-eth chain-up-usdt

chain-down-all: chain-down-usdt chain-down-eth chain-down-btc

local-up: service-up chain-up-btc

local-up-all: service-up chain-up-btc chain-up-eth chain-up-usdt

local-down: service-down chain-down-all
