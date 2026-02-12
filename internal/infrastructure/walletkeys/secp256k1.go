package walletkeys

import (
	"fmt"
	"math/big"
)

var (
	secp256k1P = mustBigHex("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f")
	secp256k1N = mustBigHex("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141")
	secp256k1G = curvePoint{
		x: mustBigHex("79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"),
		y: mustBigHex("483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8"),
	}
	secp256k1SqrtExponent = new(big.Int).Div(new(big.Int).Add(secp256k1P, big.NewInt(1)), big.NewInt(4))
)

type curvePoint struct {
	x        *big.Int
	y        *big.Int
	infinity bool
}

func mustBigHex(value string) *big.Int {
	out := new(big.Int)
	if _, ok := out.SetString(value, 16); !ok {
		panic("invalid secp256k1 constant")
	}
	return out
}

func fieldNormalize(value *big.Int) *big.Int {
	if value == nil {
		return big.NewInt(0)
	}
	out := new(big.Int).Mod(value, secp256k1P)
	if out.Sign() < 0 {
		out.Add(out, secp256k1P)
	}
	return out
}

func fieldAdd(a, b *big.Int) *big.Int {
	return fieldNormalize(new(big.Int).Add(a, b))
}

func fieldSub(a, b *big.Int) *big.Int {
	return fieldNormalize(new(big.Int).Sub(a, b))
}

func fieldMul(a, b *big.Int) *big.Int {
	return fieldNormalize(new(big.Int).Mul(a, b))
}

func fieldInv(value *big.Int) *big.Int {
	inv := new(big.Int).ModInverse(fieldNormalize(value), secp256k1P)
	if inv == nil {
		return nil
	}
	return inv
}

func pointDouble(point curvePoint) curvePoint {
	if point.infinity || point.y.Sign() == 0 {
		return curvePoint{infinity: true}
	}

	numerator := fieldMul(big.NewInt(3), fieldMul(point.x, point.x))
	denominatorInv := fieldInv(fieldMul(big.NewInt(2), point.y))
	if denominatorInv == nil {
		return curvePoint{infinity: true}
	}
	lambda := fieldMul(numerator, denominatorInv)

	x3 := fieldSub(fieldMul(lambda, lambda), fieldMul(big.NewInt(2), point.x))
	y3 := fieldSub(fieldMul(lambda, fieldSub(point.x, x3)), point.y)
	return curvePoint{x: x3, y: y3}
}

func pointAdd(left, right curvePoint) curvePoint {
	if left.infinity {
		return right
	}
	if right.infinity {
		return left
	}

	if left.x.Cmp(right.x) == 0 {
		if fieldAdd(left.y, right.y).Sign() == 0 {
			return curvePoint{infinity: true}
		}
		return pointDouble(left)
	}

	numerator := fieldSub(right.y, left.y)
	denominatorInv := fieldInv(fieldSub(right.x, left.x))
	if denominatorInv == nil {
		return curvePoint{infinity: true}
	}
	lambda := fieldMul(numerator, denominatorInv)

	x3 := fieldSub(fieldSub(fieldMul(lambda, lambda), left.x), right.x)
	y3 := fieldSub(fieldMul(lambda, fieldSub(left.x, x3)), left.y)
	return curvePoint{x: x3, y: y3}
}

func scalarMult(point curvePoint, scalar []byte) curvePoint {
	result := curvePoint{infinity: true}
	addend := point

	for _, byteValue := range scalar {
		for bit := 7; bit >= 0; bit-- {
			result = pointDouble(result)
			if ((byteValue >> uint(bit)) & 0x01) == 1 {
				result = pointAdd(result, addend)
			}
		}
	}

	return result
}

func scalarMultBase(scalar []byte) curvePoint {
	return scalarMult(secp256k1G, scalar)
}

func parseCompressedPublicKey(raw []byte) (curvePoint, error) {
	if len(raw) != 33 {
		return curvePoint{}, fmt.Errorf("compressed public key must be 33 bytes")
	}
	if raw[0] != 0x02 && raw[0] != 0x03 {
		return curvePoint{}, fmt.Errorf("compressed public key prefix is invalid")
	}

	x := new(big.Int).SetBytes(raw[1:])
	if x.Cmp(secp256k1P) >= 0 {
		return curvePoint{}, fmt.Errorf("public key x coordinate out of range")
	}

	// y^2 = x^3 + 7 mod p
	rhs := fieldAdd(fieldMul(fieldMul(x, x), x), big.NewInt(7))
	y := new(big.Int).Exp(rhs, secp256k1SqrtExponent, secp256k1P)
	if fieldMul(y, y).Cmp(rhs) != 0 {
		return curvePoint{}, fmt.Errorf("public key is not on secp256k1 curve")
	}

	yIsOdd := y.Bit(0) == 1
	expectOdd := raw[0] == 0x03
	if yIsOdd != expectOdd {
		y = fieldSub(secp256k1P, y)
	}

	return curvePoint{x: x, y: y}, nil
}

func serializeCompressedPublicKey(point curvePoint) ([]byte, error) {
	if point.infinity {
		return nil, fmt.Errorf("cannot serialize point at infinity")
	}
	if point.x == nil || point.y == nil {
		return nil, fmt.Errorf("invalid curve point")
	}

	out := make([]byte, 33)
	if point.y.Bit(0) == 0 {
		out[0] = 0x02
	} else {
		out[0] = 0x03
	}

	xBytes := point.x.Bytes()
	copy(out[33-len(xBytes):], xBytes)
	return out, nil
}

func serializeUncompressedPublicKey(point curvePoint) ([]byte, error) {
	if point.infinity {
		return nil, fmt.Errorf("cannot serialize point at infinity")
	}
	if point.x == nil || point.y == nil {
		return nil, fmt.Errorf("invalid curve point")
	}

	out := make([]byte, 65)
	out[0] = 0x04
	xBytes := point.x.Bytes()
	yBytes := point.y.Bytes()
	copy(out[33-len(xBytes):33], xBytes)
	copy(out[65-len(yBytes):], yBytes)
	return out, nil
}
