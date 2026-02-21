package devtest

type confirmationPolicy struct {
	btcMin int
	evmMin int
}

func newConfirmationPolicy(
	rawBTC int,
	rawEVM int,
	defaultBTC int,
	defaultEVM int,
) confirmationPolicy {
	btcMin := rawBTC
	if btcMin <= 0 {
		btcMin = defaultBTC
	}

	evmMin := rawEVM
	if evmMin <= 0 {
		evmMin = defaultEVM
	}

	return confirmationPolicy{
		btcMin: btcMin,
		evmMin: evmMin,
	}
}
