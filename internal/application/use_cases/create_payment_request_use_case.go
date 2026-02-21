package use_cases

import (
	"context"
	"strings"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	valueobjects "chaintx/internal/domain/value_objects"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type createPaymentRequestUseCase struct {
	assetCatalogReadModel portsout.AssetCatalogReadModel
	repository            portsout.PaymentRequestRepository
	walletGateway         portsout.WalletAllocationGateway
	allocationMode        string
	webhookURLAllowList   []string
	clock                 Clock
}

func NewCreatePaymentRequestUseCase(
	assetCatalogReadModel portsout.AssetCatalogReadModel,
	repository portsout.PaymentRequestRepository,
	walletGateway portsout.WalletAllocationGateway,
	clock Clock,
	webhookURLAllowList []string,
) portsin.CreatePaymentRequestUseCase {
	if clock == nil {
		clock = NewSystemClock()
	}

	return &createPaymentRequestUseCase{
		assetCatalogReadModel: assetCatalogReadModel,
		repository:            repository,
		walletGateway:         walletGateway,
		allocationMode:        detectAllocationMode(walletGateway),
		webhookURLAllowList:   webhookURLAllowList,
		clock:                 clock,
	}
}

func (u *createPaymentRequestUseCase) Execute(ctx context.Context, command dto.CreatePaymentRequestCommand) (dto.CreatePaymentRequestOutput, *apperrors.AppError) {
	if appErr := u.validateDependencies(); appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}

	normalizedInput, appErr := u.normalizeCommand(command)
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}

	assetEntry, appErr := u.loadAssetCatalogEntry(ctx, normalizedInput.Chain, normalizedInput.Network, normalizedInput.Asset)
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}

	persistenceCommand, appErr := u.buildPersistenceCommand(normalizedInput, assetEntry)
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}

	result, appErr := u.repository.Create(ctx, persistenceCommand, u.resolvePaymentAddress)
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}

	return dto.CreatePaymentRequestOutput(result), nil
}

func (u *createPaymentRequestUseCase) resolvePaymentAddress(
	ctx context.Context,
	input dto.ResolvePaymentAddressInput,
) (dto.ResolvePaymentAddressOutput, *apperrors.AppError) {
	derived, appErr := u.walletGateway.DeriveAddress(ctx, portsout.DeriveAddressInput{
		Chain:                  input.Chain,
		Network:                input.Network,
		AddressScheme:          input.AddressScheme,
		KeysetID:               input.KeysetID,
		DerivationPathTemplate: input.DerivationPathTemplate,
		DerivationIndex:        input.DerivationIndex,
		ChainID:                input.ChainID,
	})
	if appErr != nil {
		return dto.ResolvePaymentAddressOutput{}, appErr
	}

	if derived.AddressScheme != "" && !strings.EqualFold(derived.AddressScheme, input.AddressScheme) {
		return dto.ResolvePaymentAddressOutput{}, apperrors.NewInternal(
			"invalid_configuration",
			"wallet gateway returned incompatible address scheme",
			map[string]any{
				"expected_address_scheme": input.AddressScheme,
				"actual_address_scheme":   derived.AddressScheme,
			},
		)
	}

	if input.ChainID != nil && derived.ChainID != nil && *input.ChainID != *derived.ChainID {
		return dto.ResolvePaymentAddressOutput{}, apperrors.NewInternal(
			"invalid_configuration",
			"wallet gateway returned incompatible chain id",
			map[string]any{
				"expected_chain_id": *input.ChainID,
				"actual_chain_id":   *derived.ChainID,
			},
		)
	}

	addressCanonical, appErr := valueobjects.NormalizeAddressForStorage(input.Chain, derived.AddressRaw)
	if appErr != nil {
		return dto.ResolvePaymentAddressOutput{}, appErr
	}

	addressResponse, appErr := valueobjects.FormatAddressForResponse(input.Chain, addressCanonical)
	if appErr != nil {
		return dto.ResolvePaymentAddressOutput{}, appErr
	}

	return dto.ResolvePaymentAddressOutput{
		AddressCanonical: addressCanonical,
		Address:          addressResponse,
	}, nil
}

type allocationModeProvider interface {
	Mode() string
}

func detectAllocationMode(gateway portsout.WalletAllocationGateway) string {
	if gateway == nil {
		return "unknown"
	}

	modeProvider, ok := gateway.(allocationModeProvider)
	if !ok {
		return "unknown"
	}

	mode := strings.ToLower(strings.TrimSpace(modeProvider.Mode()))
	if mode == "" {
		return "unknown"
	}

	return mode
}
