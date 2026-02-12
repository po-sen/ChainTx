package walletkeys

import "golang.org/x/crypto/sha3"

func legacyKeccak256(input []byte) [32]byte {
	hash := sha3.NewLegacyKeccak256()
	_, _ = hash.Write(input)
	sum := hash.Sum(nil)

	var out [32]byte
	copy(out[:], sum[:32])
	return out
}
