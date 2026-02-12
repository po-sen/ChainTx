package walletkeys

import (
	"fmt"
	"strings"
)

const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

var bech32Generator = [5]uint32{
	0x3b6a57b2,
	0x26508e6d,
	0x1ea119fa,
	0x3d4233dd,
	0x2a1462b3,
}

func bech32Polymod(values []byte) uint32 {
	chk := uint32(1)
	for _, value := range values {
		top := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ uint32(value)
		for i := 0; i < len(bech32Generator); i++ {
			if ((top >> uint(i)) & 1) == 1 {
				chk ^= bech32Generator[i]
			}
		}
	}
	return chk
}

func bech32HRPExpand(hrp string) []byte {
	expanded := make([]byte, 0, len(hrp)*2+1)
	for i := 0; i < len(hrp); i++ {
		expanded = append(expanded, hrp[i]>>5)
	}
	expanded = append(expanded, 0)
	for i := 0; i < len(hrp); i++ {
		expanded = append(expanded, hrp[i]&31)
	}
	return expanded
}

func bech32CreateChecksum(hrp string, data []byte) []byte {
	values := append(bech32HRPExpand(hrp), data...)
	values = append(values, 0, 0, 0, 0, 0, 0)
	polymod := bech32Polymod(values) ^ 1

	checksum := make([]byte, 6)
	for i := 0; i < 6; i++ {
		checksum[i] = byte((polymod >> uint(5*(5-i))) & 31)
	}
	return checksum
}

func bech32Encode(hrp string, data []byte) (string, error) {
	if hrp == "" {
		return "", fmt.Errorf("bech32 hrp is empty")
	}

	hrp = strings.ToLower(hrp)
	checksum := bech32CreateChecksum(hrp, data)
	combined := append(data, checksum...)

	builder := strings.Builder{}
	builder.Grow(len(hrp) + 1 + len(combined))
	builder.WriteString(hrp)
	builder.WriteByte('1')
	for _, value := range combined {
		if value >= 32 {
			return "", fmt.Errorf("bech32 value out of range")
		}
		builder.WriteByte(bech32Charset[value])
	}

	return builder.String(), nil
}

func convertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	var (
		accumulator uint
		bits        uint
		maxValue    = uint((1 << toBits) - 1)
	)
	out := make([]byte, 0, len(data)*int(fromBits)/int(toBits)+1)

	for _, value := range data {
		if uint(value)>>fromBits != 0 {
			return nil, fmt.Errorf("value out of range")
		}
		accumulator = (accumulator << fromBits) | uint(value)
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			out = append(out, byte((accumulator>>bits)&maxValue))
		}
	}

	if pad {
		if bits > 0 {
			out = append(out, byte((accumulator<<(toBits-bits))&maxValue))
		}
	} else if bits >= fromBits || ((accumulator<<(toBits-bits))&maxValue) != 0 {
		return nil, fmt.Errorf("invalid padding")
	}

	return out, nil
}

func encodeSegWitAddress(hrp string, witnessVersion byte, witnessProgram []byte) (string, error) {
	if witnessVersion > 16 {
		return "", fmt.Errorf("witness version out of range")
	}

	converted, err := convertBits(witnessProgram, 8, 5, true)
	if err != nil {
		return "", err
	}

	data := make([]byte, 0, 1+len(converted))
	data = append(data, witnessVersion)
	data = append(data, converted...)
	return bech32Encode(hrp, data)
}
