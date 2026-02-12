//go:build !integration

package walletkeys

import (
	"strings"
	"testing"
)

func TestNormalizeBitcoinKeysetAcceptsTPubAndVPub(t *testing.T) {
	tpub := testExtendedPublicKey(versionTPub).Serialize()
	normalizedTPub, _, keyErr := NormalizeBitcoinKeyset(tpub)
	if keyErr != nil {
		t.Fatalf("expected tpub to normalize, got %+v", keyErr)
	}
	if normalizedTPub.Version != versionTPub {
		t.Fatalf("expected normalized version tpub, got %#x", normalizedTPub.Version)
	}

	vpub := testExtendedPublicKey(versionVPub).Serialize()
	normalizedVPub, serialized, keyErr := NormalizeBitcoinKeyset(vpub)
	if keyErr != nil {
		t.Fatalf("expected vpub to normalize, got %+v", keyErr)
	}
	if normalizedVPub.Version != versionTPub {
		t.Fatalf("expected normalized vpub version tpub, got %#x", normalizedVPub.Version)
	}
	if _, parseErr := ParseExtendedPublicKey(serialized); parseErr != nil {
		t.Fatalf("expected normalized serialized key to parse, got %+v", parseErr)
	}
}

func TestNormalizeEVMKeysetAcceptsXPubAndTPubRejectsVPub(t *testing.T) {
	xpub := testExtendedPublicKey(versionXPub).Serialize()
	normalizedXPub, _, keyErr := NormalizeEVMKeyset(xpub)
	if keyErr != nil {
		t.Fatalf("expected xpub to normalize, got %+v", keyErr)
	}
	if normalizedXPub.Version != versionXPub {
		t.Fatalf("expected normalized version xpub, got %#x", normalizedXPub.Version)
	}

	tpub := testExtendedPublicKey(versionTPub).Serialize()
	normalizedTPub, _, keyErr := NormalizeEVMKeyset(tpub)
	if keyErr != nil {
		t.Fatalf("expected tpub to normalize for evm, got %+v", keyErr)
	}
	if normalizedTPub.Version != versionXPub {
		t.Fatalf("expected normalized version xpub, got %#x", normalizedTPub.Version)
	}

	vpub := testExtendedPublicKey(versionVPub).Serialize()
	if _, _, keyErr := NormalizeEVMKeyset(vpub); keyErr == nil {
		t.Fatalf("expected vpub rejection for evm")
	}
}

func TestValidateAccountLevelPolicy(t *testing.T) {
	key := testExtendedPublicKey(versionTPub)
	if keyErr := ValidateAccountLevelPolicy(key); keyErr != nil {
		t.Fatalf("expected valid account-level key, got %+v", keyErr)
	}

	key.Depth = 4
	if keyErr := ValidateAccountLevelPolicy(key); keyErr == nil {
		t.Fatalf("expected depth policy rejection")
	}

	key = testExtendedPublicKey(versionTPub)
	key.ChildNumber = 7
	if keyErr := ValidateAccountLevelPolicy(key); keyErr == nil {
		t.Fatalf("expected hardened child policy rejection")
	}
}

func TestValidateDerivationPathTemplate(t *testing.T) {
	if keyErr := ValidateDerivationPathTemplate("0/{index}"); keyErr != nil {
		t.Fatalf("expected valid template, got %+v", keyErr)
	}
	if keyErr := ValidateDerivationPathTemplate("{index}"); keyErr == nil {
		t.Fatalf("expected invalid template rejection")
	}
	if keyErr := ValidateDerivationPathTemplate("0'/{index}"); keyErr == nil {
		t.Fatalf("expected hardened template rejection")
	}
}

func TestDeriveBitcoinP2WPKHAddressDeterministic(t *testing.T) {
	key := testExtendedPublicKey(versionTPub)

	regtestA, keyErr := DeriveBitcoinP2WPKHAddress(key, "regtest", "0/{index}", 0)
	if keyErr != nil {
		t.Fatalf("expected regtest derivation success, got %+v", keyErr)
	}
	regtestB, keyErr := DeriveBitcoinP2WPKHAddress(key, "regtest", "0/{index}", 0)
	if keyErr != nil {
		t.Fatalf("expected regtest derivation success, got %+v", keyErr)
	}
	if regtestA != regtestB {
		t.Fatalf("expected deterministic regtest derivation, got %s vs %s", regtestA, regtestB)
	}
	if !strings.HasPrefix(regtestA, "bcrt1") {
		t.Fatalf("expected regtest bech32 prefix bcrt1, got %s", regtestA)
	}

	testnetAddress, keyErr := DeriveBitcoinP2WPKHAddress(key, "testnet", "0/{index}", 1)
	if keyErr != nil {
		t.Fatalf("expected testnet derivation success, got %+v", keyErr)
	}
	if !strings.HasPrefix(testnetAddress, "tb1") {
		t.Fatalf("expected testnet bech32 prefix tb1, got %s", testnetAddress)
	}
}

func TestDeriveEVMAddressDeterministic(t *testing.T) {
	key := testExtendedPublicKey(versionXPub)

	first, keyErr := DeriveEVMAddress(key, "0/{index}", 42)
	if keyErr != nil {
		t.Fatalf("expected evm derivation success, got %+v", keyErr)
	}
	second, keyErr := DeriveEVMAddress(key, "0/{index}", 42)
	if keyErr != nil {
		t.Fatalf("expected evm derivation success, got %+v", keyErr)
	}
	if first != second {
		t.Fatalf("expected deterministic evm derivation, got %s vs %s", first, second)
	}
	if !strings.HasPrefix(first, "0x") || len(first) != 42 {
		t.Fatalf("expected 20-byte evm address, got %s", first)
	}
}

func testExtendedPublicKey(version uint32) ExtendedPublicKey {
	publicKey, err := serializeCompressedPublicKey(secp256k1G)
	if err != nil {
		panic(err)
	}

	key := ExtendedPublicKey{
		Version:           version,
		Depth:             3,
		ParentFingerprint: 0x01020304,
		ChildNumber:       0x80000002,
	}
	for i := 0; i < len(key.ChainCode); i++ {
		key.ChainCode[i] = byte(i + 1)
	}
	copy(key.PublicKeyCompressed[:], publicKey)
	return key
}
