package walletkeys

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
)

const (
	versionXPub uint32 = 0x0488b21e
	versionTPub uint32 = 0x043587cf
	versionVPub uint32 = 0x045f1cf6
)

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

var (
	bigZero       = big.NewInt(0)
	bigFiftyEight = big.NewInt(58)
)

func decodeBase58(input string) ([]byte, error) {
	value := big.NewInt(0)

	for i := 0; i < len(input); i++ {
		ch := input[i]
		index := int64(-1)
		for j := 0; j < len(base58Alphabet); j++ {
			if base58Alphabet[j] == ch {
				index = int64(j)
				break
			}
		}
		if index < 0 {
			return nil, fmt.Errorf("invalid base58 character: %q", ch)
		}

		value.Mul(value, bigFiftyEight)
		value.Add(value, big.NewInt(index))
	}

	decoded := value.Bytes()
	leadingZeroes := 0
	for leadingZeroes < len(input) && input[leadingZeroes] == '1' {
		leadingZeroes++
	}

	out := make([]byte, leadingZeroes+len(decoded))
	copy(out[leadingZeroes:], decoded)
	return out, nil
}

func encodeBase58(input []byte) string {
	value := new(big.Int).SetBytes(input)
	mod := new(big.Int)
	encoded := make([]byte, 0)

	for value.Cmp(bigZero) > 0 {
		value.DivMod(value, bigFiftyEight, mod)
		encoded = append(encoded, base58Alphabet[mod.Int64()])
	}

	for i := 0; i < len(input) && input[i] == 0; i++ {
		encoded = append(encoded, '1')
	}

	for i, j := 0, len(encoded)-1; i < j; i, j = i+1, j-1 {
		encoded[i], encoded[j] = encoded[j], encoded[i]
	}

	return string(encoded)
}

func decodeBase58Check(input string) ([]byte, error) {
	decoded, err := decodeBase58(input)
	if err != nil {
		return nil, err
	}
	if len(decoded) < 4 {
		return nil, fmt.Errorf("base58check payload too short")
	}

	payload := decoded[:len(decoded)-4]
	checksum := decoded[len(decoded)-4:]
	expected := checksum4(payload)
	if checksum[0] != expected[0] || checksum[1] != expected[1] || checksum[2] != expected[2] || checksum[3] != expected[3] {
		return nil, fmt.Errorf("base58check checksum mismatch")
	}

	return payload, nil
}

func encodeBase58Check(payload []byte) string {
	checksum := checksum4(payload)
	buffer := make([]byte, 0, len(payload)+4)
	buffer = append(buffer, payload...)
	buffer = append(buffer, checksum[:]...)
	return encodeBase58(buffer)
}

func checksum4(payload []byte) [4]byte {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])
	return [4]byte{second[0], second[1], second[2], second[3]}
}

func readVersion(payload []byte) uint32 {
	return binary.BigEndian.Uint32(payload[:4])
}

func writeVersion(payload []byte, version uint32) {
	binary.BigEndian.PutUint32(payload[:4], version)
}
