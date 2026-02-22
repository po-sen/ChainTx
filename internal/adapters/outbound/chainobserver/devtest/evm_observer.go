package devtest

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"strings"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

const (
	erc20TransferTopic0 = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

type evmObserver struct {
	rpcURLs       map[string]string
	rpcClient     *jsonRPCClient
	thresholds    thresholdPolicy
	confirmations confirmationPolicy
}

type evmBlock struct {
	Hash         string           `json:"hash"`
	Number       string           `json:"number"`
	Transactions []evmTransaction `json:"transactions"`
}

type evmTransaction struct {
	Hash        string `json:"hash"`
	To          string `json:"to"`
	Value       string `json:"value"`
	BlockNumber string `json:"blockNumber"`
	BlockHash   string `json:"blockHash"`
}

type evmTransactionReceipt struct {
	Status string `json:"status"`
}

type erc20TransferLog struct {
	Removed         bool   `json:"removed"`
	TransactionHash string `json:"transactionHash"`
	LogIndex        string `json:"logIndex"`
	BlockNumber     string `json:"blockNumber"`
	BlockHash       string `json:"blockHash"`
	Data            string `json:"data"`
}

func newEVMObserver(
	rawRPCURLs map[string]string,
	rpcClient *jsonRPCClient,
	thresholds thresholdPolicy,
	confirmations confirmationPolicy,
) *evmObserver {
	normalized := map[string]string{}
	for network, rawURL := range rawRPCURLs {
		normalizedNetwork := strings.ToLower(strings.TrimSpace(network))
		normalizedURL := strings.TrimSpace(rawURL)
		if normalizedNetwork == "" || normalizedURL == "" {
			continue
		}
		normalized[normalizedNetwork] = normalizedURL
	}

	return &evmObserver{
		rpcURLs:       normalized,
		rpcClient:     rpcClient,
		thresholds:    thresholds,
		confirmations: confirmations,
	}
}

func (o *evmObserver) Observe(
	ctx context.Context,
	input dto.ObservePaymentRequestInput,
	expected *big.Int,
) (dto.ObservePaymentRequestOutput, *apperrors.AppError) {
	if o == nil || o.rpcClient == nil {
		return dto.ObservePaymentRequestOutput{Supported: false}, nil
	}

	network := strings.ToLower(strings.TrimSpace(input.Network))
	rpcURL, exists := o.rpcURLs[network]
	if !exists || strings.TrimSpace(rpcURL) == "" {
		return dto.ObservePaymentRequestOutput{Supported: false}, nil
	}

	latestBlock, appErr := o.latestBlockNumber(ctx, rpcURL)
	if appErr != nil {
		return dto.ObservePaymentRequestOutput{}, appErr
	}

	asset := strings.ToUpper(strings.TrimSpace(input.Asset))
	normalizedAddress := normalizeHexAddress(input.AddressCanonical)
	// Devtest mode favors correctness over throughput and scans full canonical history.
	startBlock := big.NewInt(0)

	var settlements []dto.ObservedSettlementEvidence
	switch asset {
	case "ETH":
		settlements, appErr = o.observeETHTransfers(ctx, rpcURL, normalizedAddress, latestBlock, startBlock)
		if appErr != nil {
			return dto.ObservePaymentRequestOutput{}, appErr
		}
	default:
		if input.TokenContract == nil || strings.TrimSpace(*input.TokenContract) == "" {
			return dto.ObservePaymentRequestOutput{Supported: false}, nil
		}
		settlements, appErr = o.observeERC20Transfers(
			ctx,
			rpcURL,
			strings.TrimSpace(*input.TokenContract),
			normalizedAddress,
			latestBlock,
			startBlock,
		)
		if appErr != nil {
			return dto.ObservePaymentRequestOutput{}, appErr
		}
	}

	latestAmount, confirmedAmount, finalityAmount := o.aggregateAmounts(settlements)
	confirmedRequired := o.thresholds.confirmedRequired(expected)
	detectedRequired := o.thresholds.detectedRequired(expected)
	confirmed := confirmedAmount.Cmp(confirmedRequired) >= 0
	finalityReached := finalityAmount.Cmp(confirmedRequired) >= 0
	detected := !confirmed && latestAmount.Cmp(detectedRequired) >= 0

	return dto.ObservePaymentRequestOutput{
		Supported:         true,
		ObservedAmount:    latestAmount.String(),
		Detected:          detected,
		Confirmed:         confirmed,
		FinalityReached:   finalityReached,
		ObservationSource: "evm_rpc",
		ObservationDetails: map[string]any{
			"network":                        network,
			"asset":                          asset,
			"detected_threshold_bps":         o.thresholds.detectedBPS,
			"confirmed_threshold_bps":        o.thresholds.confirmedBPS,
			"detected_required_minor":        detectedRequired.String(),
			"confirmed_required_minor":       confirmedRequired.String(),
			"evm_business_min_confirmations": o.confirmations.evmBusinessMin,
			"evm_finality_min_confirmations": o.confirmations.evmFinalityMin,
			"latest_amount_minor":            latestAmount.String(),
			"confirmed_amount_minor":         confirmedAmount.String(),
			"finality_amount_minor":          finalityAmount.String(),
			"settlement_item_count":          len(settlements),
			"scan_start_block":               blockToDecimalString(startBlock),
			"scan_latest_block":              blockToDecimalString(latestBlock),
			"scan_scope":                     "full_history",
		},
		Settlements: settlements,
	}, nil
}

func (o *evmObserver) observeETHTransfers(
	ctx context.Context,
	rpcURL string,
	recipient string,
	latestBlock *big.Int,
	startBlock *big.Int,
) ([]dto.ObservedSettlementEvidence, *apperrors.AppError) {
	settlements := []dto.ObservedSettlementEvidence{}
	one := big.NewInt(1)

	for height := new(big.Int).Set(startBlock); height.Cmp(latestBlock) <= 0; height.Add(height, one) {
		blockTag := blockTagFromHeight(height)
		block, blockErr := o.fetchBlockWithTransactions(ctx, rpcURL, blockTag)
		if blockErr != nil {
			return nil, blockErr
		}
		if block == nil || len(block.Transactions) == 0 {
			continue
		}

		for index, tx := range block.Transactions {
			txHash := strings.ToLower(strings.TrimSpace(tx.Hash))
			if txHash == "" {
				continue
			}
			if normalizeHexAddress(tx.To) != recipient {
				continue
			}

			value, valueErr := parseHexQuantityString(tx.Value)
			if valueErr != nil {
				return nil, valueErr
			}
			if value.Sign() <= 0 {
				continue
			}

			success, receiptErr := o.transactionSucceeded(ctx, rpcURL, txHash)
			if receiptErr != nil {
				return nil, receiptErr
			}
			if !success {
				continue
			}

			blockNumber, parseErr := blockNumberFromTx(tx, block, height)
			if parseErr != nil {
				return nil, parseErr
			}
			confirmations := confirmationCount(latestBlock, blockNumber)
			heightValue := int64PointerFromBigInt(blockNumber)
			blockHash := blockHashFromTx(tx, block)
			metadata := map[string]any{
				"source":        "eth_transfer",
				"block_tag":     blockTag,
				"tx_list_index": index,
			}

			settlements = append(settlements, dto.ObservedSettlementEvidence{
				EvidenceRef:   txHash,
				AmountMinor:   value.String(),
				Confirmations: confirmations,
				IsCanonical:   true,
				BlockHeight:   heightValue,
				BlockHash:     blockHash,
				Metadata:      metadata,
			})
		}
	}

	return settlements, nil
}

func (o *evmObserver) observeERC20Transfers(
	ctx context.Context,
	rpcURL string,
	tokenContract string,
	recipient string,
	latestBlock *big.Int,
	startBlock *big.Int,
) ([]dto.ObservedSettlementEvidence, *apperrors.AppError) {
	normalizedContract := normalizeHexAddress(tokenContract)
	logs, appErr := o.fetchERC20TransferLogs(ctx, rpcURL, normalizedContract, recipient, startBlock)
	if appErr != nil {
		return nil, appErr
	}

	settlements := make([]dto.ObservedSettlementEvidence, 0, len(logs))
	for _, logEntry := range logs {
		txHash := strings.ToLower(strings.TrimSpace(logEntry.TransactionHash))
		if txHash == "" {
			continue
		}

		logIndex, logErr := parseHexQuantityString(logEntry.LogIndex)
		if logErr != nil {
			return nil, logErr
		}
		amount, amountErr := parseHexQuantityString(logEntry.Data)
		if amountErr != nil {
			return nil, amountErr
		}
		if amount.Sign() <= 0 {
			continue
		}

		blockNumber, blockErr := parseHexQuantityString(logEntry.BlockNumber)
		if blockErr != nil {
			return nil, blockErr
		}
		confirmations := confirmationCount(latestBlock, blockNumber)
		heightValue := int64PointerFromBigInt(blockNumber)
		blockHash := normalizedOptionalHex(logEntry.BlockHash)
		evidenceRef := txHash + ":" + logIndex.String()

		settlements = append(settlements, dto.ObservedSettlementEvidence{
			EvidenceRef:   evidenceRef,
			AmountMinor:   amount.String(),
			Confirmations: confirmations,
			IsCanonical:   !logEntry.Removed,
			BlockHeight:   heightValue,
			BlockHash:     blockHash,
			Metadata: map[string]any{
				"source":         "erc20_transfer",
				"token_contract": normalizedContract,
				"log_index":      logIndex.String(),
			},
		})
	}

	return settlements, nil
}

func (o *evmObserver) aggregateAmounts(settlements []dto.ObservedSettlementEvidence) (*big.Int, *big.Int, *big.Int) {
	latestAmount := big.NewInt(0)
	confirmedAmount := big.NewInt(0)
	finalityAmount := big.NewInt(0)

	for _, settlement := range settlements {
		if !settlement.IsCanonical {
			continue
		}
		amount := new(big.Int)
		if _, ok := amount.SetString(strings.TrimSpace(settlement.AmountMinor), 10); !ok || amount.Sign() < 0 {
			continue
		}

		latestAmount.Add(latestAmount, amount)
		if settlement.Confirmations >= o.confirmations.evmBusinessMin {
			confirmedAmount.Add(confirmedAmount, amount)
		}
		if settlement.Confirmations >= o.confirmations.evmFinalityMin {
			finalityAmount.Add(finalityAmount, amount)
		}
	}

	return latestAmount, confirmedAmount, finalityAmount
}

func (o *evmObserver) fetchBlockWithTransactions(
	ctx context.Context,
	rpcURL string,
	blockTag string,
) (*evmBlock, *apperrors.AppError) {
	result, appErr := o.rpcClient.Call(ctx, rpcURL, "eth_getBlockByNumber", []any{blockTag, true})
	if appErr != nil {
		return nil, appErr
	}
	trimmed := strings.TrimSpace(string(result))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	block := evmBlock{}
	if err := json.Unmarshal(result, &block); err != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to decode block payload",
			map[string]any{"error": err.Error(), "block_tag": blockTag},
		)
	}
	return &block, nil
}

func (o *evmObserver) transactionSucceeded(
	ctx context.Context,
	rpcURL string,
	txHash string,
) (bool, *apperrors.AppError) {
	result, appErr := o.rpcClient.Call(ctx, rpcURL, "eth_getTransactionReceipt", []any{txHash})
	if appErr != nil {
		return false, appErr
	}
	trimmed := strings.TrimSpace(string(result))
	if trimmed == "" || trimmed == "null" {
		return false, nil
	}

	receipt := evmTransactionReceipt{}
	if err := json.Unmarshal(result, &receipt); err != nil {
		return false, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to decode transaction receipt",
			map[string]any{"error": err.Error(), "tx_hash": txHash},
		)
	}
	status, parseErr := parseHexQuantityString(receipt.Status)
	if parseErr != nil {
		return false, parseErr
	}
	return status.Sign() > 0, nil
}

func (o *evmObserver) fetchERC20TransferLogs(
	ctx context.Context,
	rpcURL string,
	tokenContract string,
	recipient string,
	startBlock *big.Int,
) ([]erc20TransferLog, *apperrors.AppError) {
	recipientTopic := recipientTopicFromAddress(recipient)
	filter := map[string]any{
		"fromBlock": blockTagFromHeight(startBlock),
		"toBlock":   "latest",
		"address":   tokenContract,
		"topics": []any{
			erc20TransferTopic0,
			nil,
			recipientTopic,
		},
	}

	result, appErr := o.rpcClient.Call(ctx, rpcURL, "eth_getLogs", []any{filter})
	if appErr != nil {
		return nil, appErr
	}
	trimmed := strings.TrimSpace(string(result))
	if trimmed == "" || trimmed == "null" {
		return []erc20TransferLog{}, nil
	}

	logs := []erc20TransferLog{}
	if err := json.Unmarshal(result, &logs); err != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to decode transfer logs",
			map[string]any{"error": err.Error(), "contract": tokenContract},
		)
	}
	return logs, nil
}

func (o *evmObserver) latestBlockNumber(
	ctx context.Context,
	rpcURL string,
) (*big.Int, *apperrors.AppError) {
	result, appErr := o.rpcClient.Call(ctx, rpcURL, "eth_blockNumber", []any{})
	if appErr != nil {
		return nil, appErr
	}
	latestBlock, parseErr := parseHexQuantity(result)
	if parseErr != nil {
		return nil, parseErr
	}
	return latestBlock, nil
}

func blockTagFromHeight(height *big.Int) string {
	if height == nil || height.Sign() < 0 {
		return "0x0"
	}
	return "0x" + height.Text(16)
}

func blockToDecimalString(block *big.Int) string {
	if block == nil {
		return "0"
	}
	return block.String()
}

func confirmationCount(latest *big.Int, block *big.Int) int {
	if latest == nil || block == nil {
		return 0
	}
	diff := new(big.Int).Sub(latest, block)
	diff.Add(diff, big.NewInt(1))
	if diff.Sign() < 0 {
		return 0
	}
	if !diff.IsInt64() {
		return math.MaxInt
	}
	value := diff.Int64()
	if value > int64(math.MaxInt) {
		return math.MaxInt
	}
	if value < 0 {
		return 0
	}
	return int(value)
}

func parseHexQuantityString(value string) (*big.Int, *apperrors.AppError) {
	encoded, err := json.Marshal(strings.TrimSpace(value))
	if err != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to encode hex quantity",
			map[string]any{"error": err.Error()},
		)
	}
	return parseHexQuantity(encoded)
}

func blockNumberFromTx(tx evmTransaction, block *evmBlock, fallback *big.Int) (*big.Int, *apperrors.AppError) {
	if parsed, err := parseHexQuantityString(tx.BlockNumber); err == nil {
		return parsed, nil
	}
	if block != nil {
		if parsed, err := parseHexQuantityString(block.Number); err == nil {
			return parsed, nil
		}
	}
	if fallback != nil {
		return new(big.Int).Set(fallback), nil
	}
	return nil, apperrors.NewInternal(
		"chain_observation_failed",
		"unable to resolve block number for transaction",
		nil,
	)
}

func blockHashFromTx(tx evmTransaction, block *evmBlock) *string {
	if txHash := normalizedOptionalHex(tx.BlockHash); txHash != nil {
		return txHash
	}
	if block == nil {
		return nil
	}
	return normalizedOptionalHex(block.Hash)
}

func normalizedOptionalHex(raw string) *string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "0x") {
		return &trimmed
	}
	normalized := "0x" + trimmed
	return &normalized
}

func int64PointerFromBigInt(value *big.Int) *int64 {
	if value == nil || !value.IsInt64() {
		return nil
	}
	parsed := value.Int64()
	return &parsed
}

func recipientTopicFromAddress(address string) string {
	normalized := strings.TrimPrefix(normalizeHexAddress(address), "0x")
	if len(normalized) > 40 {
		normalized = normalized[len(normalized)-40:]
	}
	if len(normalized) < 40 {
		normalized = strings.Repeat("0", 40-len(normalized)) + normalized
	}
	return "0x000000000000000000000000" + normalized
}
