//go:build !integration

package bootstrap

import (
	"database/sql"
	"testing"

	"chaintx/internal/infrastructure/walletkeys"
)

const (
	testTPub = "tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"
	testXPub = "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj"
)

func TestValidateCatalogRowHappyPathBitcoin(t *testing.T) {
	gateway := newValidationTestGateway(map[string]string{
		"ks_btc_testnet": testTPub,
	}, false)

	row := catalogValidationRow{
		Chain:                   "bitcoin",
		Network:                 "testnet",
		Asset:                   "BTC",
		WalletAccountID:         "wa_btc_testnet_001",
		AddressScheme:           "bip84_p2wpkh",
		DefaultExpiresInSeconds: 3600,
		TokenStandard:           sql.NullString{},
		TokenContract:           sql.NullString{},
		TokenDecimals:           sql.NullInt64{},
		WalletID:                sql.NullString{String: "wa_btc_testnet_001", Valid: true},
		WalletChain:             sql.NullString{String: "bitcoin", Valid: true},
		WalletNetwork:           sql.NullString{String: "testnet", Valid: true},
		WalletKeysetID:          sql.NullString{String: "ks_btc_testnet", Valid: true},
		WalletPathTemplate:      sql.NullString{String: "0/{index}", Valid: true},
		WalletNextIndex:         sql.NullInt64{Int64: 0, Valid: true},
		WalletIsActive:          sql.NullBool{Bool: true, Valid: true},
	}

	if appErr := gateway.validateCatalogRow(row); appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
}

func TestValidateCatalogRowRejectsInvalidAddressScheme(t *testing.T) {
	gateway := newValidationTestGateway(map[string]string{
		"ks_btc_testnet": testTPub,
	}, false)

	row := newBitcoinValidationRow()
	row.AddressScheme = "legacy_p2pkh"

	appErr := gateway.validateCatalogRow(row)
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "invalid_configuration" {
		t.Fatalf("expected invalid_configuration, got %s", appErr.Code)
	}
}

func TestValidateCatalogRowRejectsInvalidKeyMaterialFormat(t *testing.T) {
	gateway := newValidationTestGateway(map[string]string{
		"ks_btc_testnet": "not-a-valid-xpub",
	}, false)

	row := newBitcoinValidationRow()
	appErr := gateway.validateCatalogRow(row)
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "invalid_key_material_format" {
		t.Fatalf("expected invalid_key_material_format, got %s", appErr.Code)
	}
}

func TestValidateCatalogRowRejectsDerivationTemplateMismatch(t *testing.T) {
	gateway := newValidationTestGateway(map[string]string{
		"ks_btc_testnet": testTPub,
	}, false)

	row := newBitcoinValidationRow()
	row.WalletPathTemplate = sql.NullString{String: "{index}", Valid: true}
	appErr := gateway.validateCatalogRow(row)
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "invalid_configuration" {
		t.Fatalf("expected invalid_configuration, got %s", appErr.Code)
	}
}

func TestValidateCatalogRowRejectsMainnetWhenGuardDisabled(t *testing.T) {
	gateway := newValidationTestGateway(map[string]string{
		"ks_btc_mainnet": testTPub,
	}, false)

	row := newBitcoinValidationRow()
	row.Network = "mainnet"
	row.WalletNetwork = sql.NullString{String: "mainnet", Valid: true}
	row.WalletKeysetID = sql.NullString{String: "ks_btc_mainnet", Valid: true}

	appErr := gateway.validateCatalogRow(row)
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "invalid_configuration" {
		t.Fatalf("expected invalid_configuration, got %s", appErr.Code)
	}
}

func TestValidateCatalogRowRejectsTokenMetadataInvariant(t *testing.T) {
	gateway := newValidationTestGateway(map[string]string{
		"ks_eth_sepolia": testXPub,
	}, false)

	row := catalogValidationRow{
		Chain:                   "ethereum",
		Network:                 "sepolia",
		Asset:                   "USDT",
		WalletAccountID:         "wa_eth_sepolia_001",
		AddressScheme:           "evm_bip44",
		DefaultExpiresInSeconds: 3600,
		ChainID:                 sql.NullInt64{Int64: 11155111, Valid: true},
		TokenStandard:           sql.NullString{String: "ERC20", Valid: true},
		TokenContract:           sql.NullString{}, // invalid: token row but missing token_contract
		TokenDecimals:           sql.NullInt64{Int64: 6, Valid: true},
		WalletID:                sql.NullString{String: "wa_eth_sepolia_001", Valid: true},
		WalletChain:             sql.NullString{String: "ethereum", Valid: true},
		WalletNetwork:           sql.NullString{String: "sepolia", Valid: true},
		WalletKeysetID:          sql.NullString{String: "ks_eth_sepolia", Valid: true},
		WalletPathTemplate:      sql.NullString{String: "0/{index}", Valid: true},
		WalletNextIndex:         sql.NullInt64{Int64: 0, Valid: true},
		WalletIsActive:          sql.NullBool{Bool: true, Valid: true},
	}

	appErr := gateway.validateCatalogRow(row)
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "invalid_configuration" {
		t.Fatalf("expected invalid_configuration, got %s", appErr.Code)
	}
}

func newValidationTestGateway(keysets map[string]string, allowMainnet bool) *Gateway {
	return NewGateway(
		"",
		"",
		"",
		ValidationRules{
			AllocationMode:      "devtest",
			DevtestAllowMainnet: allowMainnet,
			DevtestKeysets:      keysets,
			DevtestKeyNormalizers: map[string]DevtestKeyNormalizer{
				"bitcoin":  walletkeys.NormalizeBitcoinKeyset,
				"ethereum": walletkeys.NormalizeEVMKeyset,
			},
			AddressSchemeAllowList: map[string]map[string]struct{}{
				"bitcoin": {
					"bip84_p2wpkh": {},
				},
				"ethereum": {
					"evm_bip44": {},
				},
			},
		},
		nil,
	)
}

func newBitcoinValidationRow() catalogValidationRow {
	return catalogValidationRow{
		Chain:                   "bitcoin",
		Network:                 "testnet",
		Asset:                   "BTC",
		WalletAccountID:         "wa_btc_testnet_001",
		AddressScheme:           "bip84_p2wpkh",
		DefaultExpiresInSeconds: 3600,
		TokenStandard:           sql.NullString{},
		TokenContract:           sql.NullString{},
		TokenDecimals:           sql.NullInt64{},
		WalletID:                sql.NullString{String: "wa_btc_testnet_001", Valid: true},
		WalletChain:             sql.NullString{String: "bitcoin", Valid: true},
		WalletNetwork:           sql.NullString{String: "testnet", Valid: true},
		WalletKeysetID:          sql.NullString{String: "ks_btc_testnet", Valid: true},
		WalletPathTemplate:      sql.NullString{String: "0/{index}", Valid: true},
		WalletNextIndex:         sql.NullInt64{Int64: 0, Valid: true},
		WalletIsActive:          sql.NullBool{Bool: true, Valid: true},
	}
}
