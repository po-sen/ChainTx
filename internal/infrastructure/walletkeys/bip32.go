package walletkeys

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strings"
)

type ExtendedPublicKey struct {
	Version             uint32
	Depth               uint8
	ParentFingerprint   uint32
	ChildNumber         uint32
	ChainCode           [32]byte
	PublicKeyCompressed [33]byte
}

func ParseExtendedPublicKey(serialized string) (ExtendedPublicKey, *KeyError) {
	payload, err := decodeBase58Check(strings.TrimSpace(serialized))
	if err != nil {
		return ExtendedPublicKey{}, wrapKeyError(CodeInvalidKeyMaterialFormat, "invalid extended public key encoding", err)
	}
	if len(payload) != 78 {
		return ExtendedPublicKey{}, wrapKeyError(CodeInvalidKeyMaterialFormat, "invalid extended public key payload length", nil)
	}

	version := readVersion(payload)
	if version != versionXPub && version != versionTPub && version != versionVPub {
		return ExtendedPublicKey{}, wrapKeyError(CodeInvalidKeyMaterialFormat, "unsupported extended public key version", nil)
	}

	depth := payload[4]
	parentFingerprint := binary.BigEndian.Uint32(payload[5:9])
	childNumber := binary.BigEndian.Uint32(payload[9:13])

	keyData := payload[45:78]
	if len(keyData) != 33 || (keyData[0] != 0x02 && keyData[0] != 0x03) {
		return ExtendedPublicKey{}, wrapKeyError(CodeInvalidKeyMaterialFormat, "extended public key data is invalid", nil)
	}
	if _, err := parseCompressedPublicKey(keyData); err != nil {
		return ExtendedPublicKey{}, wrapKeyError(CodeInvalidKeyMaterialFormat, "extended public key point is invalid", err)
	}

	out := ExtendedPublicKey{
		Version:           version,
		Depth:             depth,
		ParentFingerprint: parentFingerprint,
		ChildNumber:       childNumber,
	}
	copy(out.ChainCode[:], payload[13:45])
	copy(out.PublicKeyCompressed[:], keyData)
	return out, nil
}

func (key ExtendedPublicKey) Serialize() string {
	payload := make([]byte, 78)
	writeVersion(payload, key.Version)
	payload[4] = key.Depth
	binary.BigEndian.PutUint32(payload[5:9], key.ParentFingerprint)
	binary.BigEndian.PutUint32(payload[9:13], key.ChildNumber)
	copy(payload[13:45], key.ChainCode[:])
	copy(payload[45:78], key.PublicKeyCompressed[:])
	return encodeBase58Check(payload)
}

func NormalizeBitcoinKeyset(raw string) (ExtendedPublicKey, string, *KeyError) {
	key, keyErr := ParseExtendedPublicKey(raw)
	if keyErr != nil {
		return ExtendedPublicKey{}, "", keyErr
	}

	switch key.Version {
	case versionTPub:
		return key, key.Serialize(), nil
	case versionVPub:
		key.Version = versionTPub
		return key, key.Serialize(), nil
	default:
		return ExtendedPublicKey{}, "", wrapKeyError(CodeInvalidKeyMaterialFormat, "bitcoin keyset must use tpub or vpub", nil)
	}
}

func NormalizeEVMKeyset(raw string) (ExtendedPublicKey, string, *KeyError) {
	key, keyErr := ParseExtendedPublicKey(raw)
	if keyErr != nil {
		return ExtendedPublicKey{}, "", keyErr
	}

	switch key.Version {
	case versionXPub:
		return key, key.Serialize(), nil
	case versionTPub:
		key.Version = versionXPub
		return key, key.Serialize(), nil
	default:
		return ExtendedPublicKey{}, "", wrapKeyError(CodeInvalidKeyMaterialFormat, "evm keyset must use xpub or tpub", nil)
	}
}

func ValidateAccountLevelPolicy(key ExtendedPublicKey) *KeyError {
	if key.Depth != 3 {
		return wrapKeyError(CodeInvalidConfiguration, "extended public key depth must be 3 (account-level)", nil)
	}
	if key.ChildNumber < 0x80000000 {
		return wrapKeyError(CodeInvalidConfiguration, "extended public key child number must be hardened account index", nil)
	}
	return nil
}

func ValidateDerivationPathTemplate(template string) *KeyError {
	trimmed := strings.TrimSpace(template)
	lower := strings.ToLower(trimmed)
	if strings.Contains(trimmed, "'") || strings.Contains(lower, "h") {
		return wrapKeyError(CodeInvalidConfiguration, "derivation path template must not contain hardened segments", nil)
	}
	if trimmed != "0/{index}" {
		return wrapKeyError(CodeInvalidConfiguration, "derivation path template must be exactly 0/{index}", nil)
	}
	return nil
}

func DeriveKeyByTemplate(key ExtendedPublicKey, template string, index int64) (ExtendedPublicKey, *KeyError) {
	if keyErr := ValidateDerivationPathTemplate(template); keyErr != nil {
		return ExtendedPublicKey{}, keyErr
	}
	if index < 0 {
		return ExtendedPublicKey{}, wrapKeyError(CodeInvalidConfiguration, "derivation index must be non-negative", nil)
	}
	if index > math.MaxInt32 {
		return ExtendedPublicKey{}, wrapKeyError(CodeDerivationFailed, "derivation index exceeds non-hardened BIP32 range", nil)
	}

	childZero, keyErr := deriveChildNonHardened(key, 0)
	if keyErr != nil {
		return ExtendedPublicKey{}, keyErr
	}
	return deriveChildNonHardened(childZero, uint32(index))
}

func deriveChildNonHardened(parent ExtendedPublicKey, childIndex uint32) (ExtendedPublicKey, *KeyError) {
	if childIndex >= 0x80000000 {
		return ExtendedPublicKey{}, wrapKeyError(CodeInvalidConfiguration, "hardened child derivation is not allowed for public key derivation", nil)
	}

	parentPoint, err := parseCompressedPublicKey(parent.PublicKeyCompressed[:])
	if err != nil {
		return ExtendedPublicKey{}, wrapKeyError(CodeInvalidKeyMaterialFormat, "parent public key is invalid", err)
	}

	payload := make([]byte, 37)
	copy(payload[:33], parent.PublicKeyCompressed[:])
	binary.BigEndian.PutUint32(payload[33:], childIndex)

	mac := hmac.New(sha512.New, parent.ChainCode[:])
	if _, err := mac.Write(payload); err != nil {
		return ExtendedPublicKey{}, wrapKeyError(CodeDerivationFailed, "failed to compute child derivation digest", err)
	}
	digest := mac.Sum(nil)
	il := digest[:32]
	ir := digest[32:]

	ilInt := new(big.Int).SetBytes(il)
	if ilInt.Sign() == 0 || ilInt.Cmp(secp256k1N) >= 0 {
		return ExtendedPublicKey{}, wrapKeyError(CodeDerivationFailed, "child derivation produced invalid scalar", nil)
	}

	derivedPoint := scalarMultBase(il)
	childPoint := pointAdd(parentPoint, derivedPoint)
	if childPoint.infinity {
		return ExtendedPublicKey{}, wrapKeyError(CodeDerivationFailed, "child derivation produced point at infinity", nil)
	}

	compressed, err := serializeCompressedPublicKey(childPoint)
	if err != nil {
		return ExtendedPublicKey{}, wrapKeyError(CodeDerivationFailed, "failed to serialize child public key", err)
	}

	out := ExtendedPublicKey{
		Version:           parent.Version,
		Depth:             parent.Depth + 1,
		ParentFingerprint: fingerprint(parent.PublicKeyCompressed[:]),
		ChildNumber:       childIndex,
	}
	copy(out.ChainCode[:], ir)
	copy(out.PublicKeyCompressed[:], compressed)
	return out, nil
}

func fingerprint(compressedPublicKey []byte) uint32 {
	hash := hash160(compressedPublicKey)
	return binary.BigEndian.Uint32(hash[:4])
}

func hash160(input []byte) [ripemd160Size]byte {
	sha := sha256.Sum256(input)
	return ripemd160Sum(sha[:])
}

func DeriveBitcoinP2WPKHAddress(key ExtendedPublicKey, network string, template string, index int64) (string, *KeyError) {
	if key.Version != versionTPub {
		return "", wrapKeyError(CodeInvalidKeyMaterialFormat, "bitcoin derivation expects normalized tpub key", nil)
	}

	child, keyErr := DeriveKeyByTemplate(key, template, index)
	if keyErr != nil {
		return "", keyErr
	}

	pubKeyHash := hash160(child.PublicKeyCompressed[:])
	hrp, keyErr := bitcoinHRP(network)
	if keyErr != nil {
		return "", keyErr
	}
	address, err := encodeSegWitAddress(hrp, 0, pubKeyHash[:])
	if err != nil {
		return "", wrapKeyError(CodeDerivationFailed, "failed to encode segwit address", err)
	}
	return strings.ToLower(address), nil
}

func DeriveEVMAddress(key ExtendedPublicKey, template string, index int64) (string, *KeyError) {
	if key.Version != versionXPub {
		return "", wrapKeyError(CodeInvalidKeyMaterialFormat, "evm derivation expects normalized xpub key", nil)
	}

	child, keyErr := DeriveKeyByTemplate(key, template, index)
	if keyErr != nil {
		return "", keyErr
	}

	point, err := parseCompressedPublicKey(child.PublicKeyCompressed[:])
	if err != nil {
		return "", wrapKeyError(CodeDerivationFailed, "failed to parse derived public key", err)
	}
	uncompressed, err := serializeUncompressedPublicKey(point)
	if err != nil {
		return "", wrapKeyError(CodeDerivationFailed, "failed to serialize uncompressed public key", err)
	}

	digest := legacyKeccak256(uncompressed[1:])
	return fmt.Sprintf("0x%x", digest[12:]), nil
}

func bitcoinHRP(network string) (string, *KeyError) {
	switch strings.ToLower(strings.TrimSpace(network)) {
	case "regtest":
		return "bcrt", nil
	case "testnet":
		return "tb", nil
	case "mainnet":
		return "bc", nil
	default:
		return "", wrapKeyError(CodeUnsupportedTarget, "unsupported bitcoin network", nil)
	}
}
