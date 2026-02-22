package devtest

type confirmationPolicy struct {
	btcBusinessMin int
	btcFinalityMin int
	evmBusinessMin int
	evmFinalityMin int
}

func newConfirmationPolicy(
	rawBTCBusiness int,
	rawBTCFinality int,
	rawEVMBusiness int,
	rawEVMFinality int,
	defaultBTC int,
	defaultEVM int,
) confirmationPolicy {
	btcBusinessMin := rawBTCBusiness
	if btcBusinessMin <= 0 {
		btcBusinessMin = defaultBTC
	}

	btcFinalityMin := rawBTCFinality
	if btcFinalityMin <= 0 {
		btcFinalityMin = btcBusinessMin
	}
	if btcFinalityMin < btcBusinessMin {
		btcFinalityMin = btcBusinessMin
	}

	evmBusinessMin := rawEVMBusiness
	if evmBusinessMin <= 0 {
		evmBusinessMin = defaultEVM
	}

	evmFinalityMin := rawEVMFinality
	if evmFinalityMin <= 0 {
		evmFinalityMin = evmBusinessMin
	}
	if evmFinalityMin < evmBusinessMin {
		evmFinalityMin = evmBusinessMin
	}

	return confirmationPolicy{
		btcBusinessMin: btcBusinessMin,
		btcFinalityMin: btcFinalityMin,
		evmBusinessMin: evmBusinessMin,
		evmFinalityMin: evmFinalityMin,
	}
}
