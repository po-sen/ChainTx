//go:build !integration

package devtest

import (
	"context"
	"strings"
	"testing"

	portsout "chaintx/internal/application/ports/out"
)

const (
	testTPub = "tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"
	testVPub = "vpub5Xzfrm6ouSBPKVriRpkXyai4mvsHjRHq28wxS1znBCdwzLzeJUx8ruJeBnCMKs1AyqYsJ2mriQHuzxNoFtkkq94J4bJyNjGXkbZ8vCYwUy3"
	testXPub = "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj"
)

func TestGatewayDeriveAddressBitcoinTestnetDeterministic(t *testing.T) {
	gateway := NewGateway(Config{
		AllowMainnet: false,
		Keysets: map[string]string{
			"ks_btc_testnet": testVPub,
		},
	}, nil)

	input := portsout.DeriveAddressInput{
		Chain:                  "bitcoin",
		Network:                "testnet",
		AddressScheme:          "bip84_p2wpkh",
		KeysetID:               "ks_btc_testnet",
		DerivationPathTemplate: "0/{index}",
		DerivationIndex:        42,
	}

	first, appErr := gateway.DeriveAddress(context.Background(), input)
	if appErr != nil {
		t.Fatalf("expected derive success, got %+v", appErr)
	}
	second, appErr := gateway.DeriveAddress(context.Background(), input)
	if appErr != nil {
		t.Fatalf("expected derive success, got %+v", appErr)
	}

	if first.AddressRaw != second.AddressRaw {
		t.Fatalf("expected deterministic derivation, got %s vs %s", first.AddressRaw, second.AddressRaw)
	}
	if !strings.HasPrefix(first.AddressRaw, "tb1") {
		t.Fatalf("expected testnet bech32 address, got %s", first.AddressRaw)
	}
}

func TestGatewayDeriveAddressMainnetBlockedByDefault(t *testing.T) {
	gateway := NewGateway(Config{
		AllowMainnet: false,
		Keysets: map[string]string{
			"ks_btc_mainnet": testTPub,
		},
	}, nil)

	_, appErr := gateway.DeriveAddress(context.Background(), portsout.DeriveAddressInput{
		Chain:                  "bitcoin",
		Network:                "mainnet",
		AddressScheme:          "bip84_p2wpkh",
		KeysetID:               "ks_btc_mainnet",
		DerivationPathTemplate: "0/{index}",
		DerivationIndex:        0,
	})
	if appErr == nil {
		t.Fatalf("expected mainnet guard error")
	}
	if appErr.Code != "mainnet_allocation_blocked" {
		t.Fatalf("expected mainnet_allocation_blocked, got %s", appErr.Code)
	}
}

func TestGatewayDeriveAddressMainnetAllowedWithOverride(t *testing.T) {
	gateway := NewGateway(Config{
		AllowMainnet: true,
		Keysets: map[string]string{
			"ks_btc_mainnet": testTPub,
		},
	}, nil)

	result, appErr := gateway.DeriveAddress(context.Background(), portsout.DeriveAddressInput{
		Chain:                  "bitcoin",
		Network:                "mainnet",
		AddressScheme:          "bip84_p2wpkh",
		KeysetID:               "ks_btc_mainnet",
		DerivationPathTemplate: "0/{index}",
		DerivationIndex:        0,
	})
	if appErr != nil {
		t.Fatalf("expected mainnet derive success with override, got %+v", appErr)
	}
	if !strings.HasPrefix(strings.ToLower(result.AddressRaw), "bc1") {
		t.Fatalf("expected mainnet bech32 address prefix bc1, got %s", result.AddressRaw)
	}
}

func TestGatewayDeriveAddressEVMSepolia(t *testing.T) {
	gateway := NewGateway(Config{
		AllowMainnet: false,
		Keysets: map[string]string{
			"ks_eth_sepolia": testXPub,
		},
	}, nil)

	chainID := int64(11155111)
	result, appErr := gateway.DeriveAddress(context.Background(), portsout.DeriveAddressInput{
		Chain:                  "ethereum",
		Network:                "sepolia",
		AddressScheme:          "evm_bip44",
		KeysetID:               "ks_eth_sepolia",
		DerivationPathTemplate: "0/{index}",
		DerivationIndex:        7,
		ChainID:                &chainID,
	})
	if appErr != nil {
		t.Fatalf("expected derive success, got %+v", appErr)
	}
	if !strings.HasPrefix(result.AddressRaw, "0x") || len(result.AddressRaw) != 42 {
		t.Fatalf("unexpected evm address: %s", result.AddressRaw)
	}
	if result.ChainID == nil || *result.ChainID != 11155111 {
		t.Fatalf("expected sepolia chain id 11155111, got %+v", result.ChainID)
	}
}

func TestGatewayDeriveAddressEVMUsesInputChainID(t *testing.T) {
	gateway := NewGateway(Config{
		AllowMainnet: false,
		Keysets: map[string]string{
			"ks_eth_local": testXPub,
		},
	}, nil)

	chainID := int64(31337)
	result, appErr := gateway.DeriveAddress(context.Background(), portsout.DeriveAddressInput{
		Chain:                  "ethereum",
		Network:                "local",
		AddressScheme:          "evm_bip44",
		KeysetID:               "ks_eth_local",
		DerivationPathTemplate: "0/{index}",
		DerivationIndex:        7,
		ChainID:                &chainID,
	})
	if appErr != nil {
		t.Fatalf("expected derive success, got %+v", appErr)
	}
	if result.ChainID == nil || *result.ChainID != chainID {
		t.Fatalf("expected local chain id %d, got %+v", chainID, result.ChainID)
	}
}

func TestGatewayDeriveAddressEVMLocalRequiresChainID(t *testing.T) {
	gateway := NewGateway(Config{
		AllowMainnet: false,
		Keysets: map[string]string{
			"ks_eth_local": testXPub,
		},
	}, nil)

	_, appErr := gateway.DeriveAddress(context.Background(), portsout.DeriveAddressInput{
		Chain:                  "ethereum",
		Network:                "local",
		AddressScheme:          "evm_bip44",
		KeysetID:               "ks_eth_local",
		DerivationPathTemplate: "0/{index}",
		DerivationIndex:        7,
	})
	if appErr == nil {
		t.Fatalf("expected missing chain id error")
	}
	if appErr.Code != "invalid_configuration" {
		t.Fatalf("expected invalid_configuration, got %s", appErr.Code)
	}
}

func TestGatewayDeriveAddressEVMSepoliaRequiresChainID(t *testing.T) {
	gateway := NewGateway(Config{
		AllowMainnet: false,
		Keysets: map[string]string{
			"ks_eth_sepolia": testXPub,
		},
	}, nil)

	_, appErr := gateway.DeriveAddress(context.Background(), portsout.DeriveAddressInput{
		Chain:                  "ethereum",
		Network:                "sepolia",
		AddressScheme:          "evm_bip44",
		KeysetID:               "ks_eth_sepolia",
		DerivationPathTemplate: "0/{index}",
		DerivationIndex:        7,
	})
	if appErr == nil {
		t.Fatalf("expected missing chain id error")
	}
	if appErr.Code != "invalid_configuration" {
		t.Fatalf("expected invalid_configuration, got %s", appErr.Code)
	}
}
