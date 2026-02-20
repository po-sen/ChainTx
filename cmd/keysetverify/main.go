package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"chaintx/internal/infrastructure/walletkeys"
)

const (
	bitcoinChain  = "bitcoin"
	ethereumChain = "ethereum"

	bitcoinAddressScheme = "bip84_p2wpkh"
	evmAddressScheme     = "evm_bip44"
)

type verifyInput struct {
	Chain             string
	Network           string
	AddressScheme     string
	KeysetID          string
	ExtendedPublicKey string
	ExpectedAddress   string
}

type verifyResult struct {
	Match           bool   `json:"match"`
	Chain           string `json:"chain"`
	Network         string `json:"network"`
	AddressScheme   string `json:"address_scheme"`
	KeysetID        string `json:"keyset_id,omitempty"`
	DerivationIndex int64  `json:"derivation_index"`
	ExpectedAddress string `json:"expected_address"`
	DerivedAddress  string `json:"derived_address"`
	Reason          string `json:"reason,omitempty"`
	ErrorCode       string `json:"error_code,omitempty"`
}

func main() {
	var input verifyInput
	flag.StringVar(&input.Chain, "chain", "", "chain (bitcoin|ethereum)")
	flag.StringVar(&input.Network, "network", "", "network (bitcoin: regtest|testnet|mainnet; ethereum: local|sepolia|mainnet)")
	flag.StringVar(&input.AddressScheme, "address-scheme", "", "address scheme (bitcoin: bip84_p2wpkh; ethereum: evm_bip44)")
	flag.StringVar(&input.KeysetID, "keyset-id", "", "optional keyset id label")
	flag.StringVar(&input.ExtendedPublicKey, "extended-public-key", "", "extended public key material (xpub/tpub/vpub)")
	flag.StringVar(&input.ExpectedAddress, "expected-address", "", "expected address for derivation index 0")
	flag.Parse()

	result, exitCode := verifyIndexZero(input)
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "{\"match\":false,\"reason\":\"failed to encode result\",\"error_code\":\"result_encode_failed\"}\n")
		os.Exit(2)
	}

	fmt.Println(string(encoded))
	os.Exit(exitCode)
}

func verifyIndexZero(input verifyInput) (verifyResult, int) {
	chain := strings.ToLower(strings.TrimSpace(input.Chain))
	network := strings.ToLower(strings.TrimSpace(input.Network))
	addressScheme := strings.ToLower(strings.TrimSpace(input.AddressScheme))
	extendedPublicKey := strings.TrimSpace(input.ExtendedPublicKey)
	expectedAddress := strings.TrimSpace(input.ExpectedAddress)

	result := verifyResult{
		Match:           false,
		Chain:           chain,
		Network:         network,
		AddressScheme:   addressScheme,
		KeysetID:        strings.TrimSpace(input.KeysetID),
		DerivationIndex: 0,
		ExpectedAddress: expectedAddress,
		DerivedAddress:  "",
	}

	if chain == "" || network == "" || addressScheme == "" || extendedPublicKey == "" || expectedAddress == "" {
		result.Reason = "missing required fields: chain, network, address-scheme, extended-public-key, expected-address"
		result.ErrorCode = "invalid_input"
		return result, 2
	}

	derivedAddress, keyErr := deriveAddress(chain, network, addressScheme, extendedPublicKey)
	if keyErr != nil {
		result.Reason = keyErr.Message
		result.ErrorCode = mapKeyErrorCode(keyErr)
		return result, 2
	}

	normalizedExpected := normalizeAddressForCompare(chain, expectedAddress)
	normalizedDerived := normalizeAddressForCompare(chain, derivedAddress)
	result.DerivedAddress = normalizedDerived

	if normalizedExpected != normalizedDerived {
		result.Reason = "derived index-0 address does not match expected address"
		result.ErrorCode = "address_mismatch"
		return result, 3
	}

	result.Match = true
	return result, 0
}

func deriveAddress(chain string, network string, addressScheme string, rawKey string) (string, *walletkeys.KeyError) {
	switch chain {
	case bitcoinChain:
		if addressScheme != bitcoinAddressScheme {
			return "", newKeyError(walletkeys.CodeInvalidConfiguration, "bitcoin address scheme must be bip84_p2wpkh")
		}
		key, _, keyErr := walletkeys.NormalizeBitcoinKeyset(rawKey)
		if keyErr != nil {
			return "", keyErr
		}
		if keyErr := walletkeys.ValidateAccountLevelPolicy(key); keyErr != nil {
			return "", keyErr
		}
		return walletkeys.DeriveBitcoinP2WPKHAddress(key, network, "0/{index}", 0)
	case ethereumChain:
		if addressScheme != evmAddressScheme {
			return "", newKeyError(walletkeys.CodeInvalidConfiguration, "ethereum address scheme must be evm_bip44")
		}
		switch network {
		case "sepolia", "local", "mainnet":
		default:
			return "", newKeyError(walletkeys.CodeUnsupportedTarget, "unsupported ethereum network")
		}
		key, _, keyErr := walletkeys.NormalizeEVMKeyset(rawKey)
		if keyErr != nil {
			return "", keyErr
		}
		if keyErr := walletkeys.ValidateAccountLevelPolicy(key); keyErr != nil {
			return "", keyErr
		}
		return walletkeys.DeriveEVMAddress(key, "0/{index}", 0)
	default:
		return "", newKeyError(walletkeys.CodeUnsupportedTarget, "unsupported chain")
	}
}

func normalizeAddressForCompare(chain string, address string) string {
	trimmed := strings.TrimSpace(address)
	switch chain {
	case bitcoinChain:
		return strings.ToLower(trimmed)
	case ethereumChain:
		return strings.ToLower(trimmed)
	default:
		return strings.ToLower(trimmed)
	}
}

func mapKeyErrorCode(keyErr *walletkeys.KeyError) string {
	if keyErr == nil {
		return "address_derivation_failed"
	}

	switch keyErr.Code {
	case walletkeys.CodeInvalidKeyMaterialFormat:
		return "invalid_key_material_format"
	case walletkeys.CodeInvalidConfiguration:
		return "invalid_configuration"
	case walletkeys.CodeUnsupportedTarget:
		return "unsupported_target"
	case walletkeys.CodeDerivationFailed:
		return "address_derivation_failed"
	default:
		return "address_derivation_failed"
	}
}

func newKeyError(code walletkeys.ErrorCode, message string) *walletkeys.KeyError {
	return &walletkeys.KeyError{
		Code:    code,
		Message: message,
	}
}
