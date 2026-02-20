package devtest

import "math/big"

const thresholdDenominatorBPS int64 = 10000

type thresholdPolicy struct {
	detectedBPS  int64
	confirmedBPS int64
}

func newThresholdPolicy(
	rawDetected int,
	rawConfirmed int,
	defaultDetected int,
	defaultConfirmed int,
) thresholdPolicy {
	detected := int64(rawDetected)
	confirmed := int64(rawConfirmed)
	if detected <= 0 {
		detected = int64(defaultDetected)
	}
	if confirmed <= 0 {
		confirmed = int64(defaultConfirmed)
	}
	if confirmed > thresholdDenominatorBPS {
		confirmed = thresholdDenominatorBPS
	}
	if detected > confirmed {
		detected = confirmed
	}

	return thresholdPolicy{
		detectedBPS:  detected,
		confirmedBPS: confirmed,
	}
}

func (p thresholdPolicy) detectedRequired(expected *big.Int) *big.Int {
	return thresholdBPSAmount(expected, p.detectedBPS)
}

func (p thresholdPolicy) confirmedRequired(expected *big.Int) *big.Int {
	return thresholdBPSAmount(expected, p.confirmedBPS)
}

func thresholdBPSAmount(expected *big.Int, bps int64) *big.Int {
	if expected == nil {
		return big.NewInt(0)
	}

	product := new(big.Int).Mul(expected, big.NewInt(bps))
	quotient, remainder := new(big.Int).QuoRem(product, big.NewInt(thresholdDenominatorBPS), new(big.Int))
	if remainder.Sign() > 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	return quotient
}
