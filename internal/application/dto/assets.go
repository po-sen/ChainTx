package dto

type ListAssetsQuery struct{}

type ListAssetsOutput struct {
	Assets []AssetCatalogEntry `json:"assets"`
}

type AssetCatalogEntry struct {
	Chain                   string  `json:"chain"`
	Network                 string  `json:"network"`
	Asset                   string  `json:"asset"`
	MinorUnit               string  `json:"minor_unit"`
	Decimals                int     `json:"decimals"`
	AddressScheme           string  `json:"address_scheme"`
	DefaultExpiresInSeconds int64   `json:"default_expires_in_seconds"`
	ChainID                 *int64  `json:"chain_id,omitempty"`
	TokenStandard           *string `json:"token_standard,omitempty"`
	TokenContract           *string `json:"token_contract,omitempty"`
	TokenDecimals           *int    `json:"token_decimals,omitempty"`

	WalletAccountID string `json:"-"`
}
