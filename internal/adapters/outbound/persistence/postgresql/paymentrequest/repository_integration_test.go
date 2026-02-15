//go:build integration

package paymentrequest

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	postgresqlbootstrap "chaintx/internal/adapters/outbound/persistence/postgresql/bootstrap"
	postgresqlshared "chaintx/internal/adapters/outbound/persistence/postgresql/shared"
	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	integrationScopePrincipal = "integration-test"
	integrationScopeMethod    = "POST"
	integrationScopePath      = "/v1/payment-requests"

	integrationHashAlgorithm = "sha256"
	integrationStatusPending = "pending"
	integrationModeDevtest   = "devtest"

	integrationRequestTTL     = 1 * time.Hour
	integrationIdempotencyTTL = 25 * time.Hour
)

type repositoryIntegrationHarness struct {
	db         *sql.DB
	repository *Repository
}

func TestPaymentRequestRepositoryCreateIntegrationSmokeByAsset(t *testing.T) {
	harness := newRepositoryIntegrationHarness(t)

	testCases := []struct {
		chain   string
		network string
		asset   string
	}{
		{chain: "bitcoin", network: "regtest", asset: "BTC"},
		{chain: "ethereum", network: "sepolia", asset: "ETH"},
		{chain: "ethereum", network: "sepolia", asset: "USDT"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("%s-%s-%s", testCase.chain, testCase.network, testCase.asset), func(t *testing.T) {
			harness.resetState(t)

			catalog := harness.mustAssetCatalogEntry(t, testCase.chain, testCase.network, testCase.asset)
			resourceID := fmt.Sprintf("pr_smoke_%s_%s_%s", testCase.chain, testCase.network, strings.ToLower(testCase.asset))
			idempotencyKey := fmt.Sprintf("smoke-%s-%s-%s", testCase.chain, testCase.network, strings.ToLower(testCase.asset))
			command := newCreatePersistenceCommand(catalog, resourceID, idempotencyKey, "hash-smoke", time.Now().UTC())

			result, appErr := harness.repository.Create(context.Background(), command, deterministicResolver)
			if appErr != nil {
				t.Fatalf("expected create success, got %+v", appErr)
			}
			if result.Replayed {
				t.Fatalf("expected non-replayed result")
			}
			if result.Resource.ID != resourceID {
				t.Fatalf("expected resource id %s, got %s", resourceID, result.Resource.ID)
			}
			if result.Resource.PaymentInstructions.DerivationIndex != 0 {
				t.Fatalf("expected derivation index 0, got %d", result.Resource.PaymentInstructions.DerivationIndex)
			}

			expectedCanonical := deterministicCanonicalAddress(catalog.Chain, 0)
			storedCanonical := harness.mustAddressCanonical(t, resourceID)
			if storedCanonical != expectedCanonical {
				t.Fatalf("expected stored canonical %s, got %s", expectedCanonical, storedCanonical)
			}

			expectedResponseAddress := deterministicResponseAddress(catalog.Chain, expectedCanonical)
			if result.Resource.PaymentInstructions.Address != expectedResponseAddress {
				t.Fatalf("expected response address %s, got %s", expectedResponseAddress, result.Resource.PaymentInstructions.Address)
			}

			nextIndex := harness.mustWalletNextIndex(t, catalog.WalletAccountID)
			if nextIndex != 1 {
				t.Fatalf("expected next_index=1, got %d", nextIndex)
			}
		})
	}
}

func TestPaymentRequestRepositoryCreateIntegrationSmokeByAssetLocalEVM(t *testing.T) {
	harness := newRepositoryIntegrationHarness(t)

	testCases := []struct {
		chain   string
		network string
		asset   string
	}{
		{chain: "ethereum", network: "local", asset: "ETH"},
		{chain: "ethereum", network: "local", asset: "USDT"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("%s-%s-%s", testCase.chain, testCase.network, testCase.asset), func(t *testing.T) {
			harness.resetState(t)

			catalog := harness.mustAssetCatalogEntry(t, testCase.chain, testCase.network, testCase.asset)
			resourceID := fmt.Sprintf("pr_smoke_%s_%s_%s", testCase.chain, testCase.network, strings.ToLower(testCase.asset))
			idempotencyKey := fmt.Sprintf("smoke-%s-%s-%s", testCase.chain, testCase.network, strings.ToLower(testCase.asset))
			command := newCreatePersistenceCommand(catalog, resourceID, idempotencyKey, "hash-smoke-local", time.Now().UTC())

			result, appErr := harness.repository.Create(context.Background(), command, deterministicResolver)
			if appErr != nil {
				t.Fatalf("expected create success, got %+v", appErr)
			}
			if result.Resource.PaymentInstructions.ChainID == nil || *result.Resource.PaymentInstructions.ChainID != 31337 {
				t.Fatalf("expected local chain id 31337, got %+v", result.Resource.PaymentInstructions.ChainID)
			}
		})
	}
}

func TestPaymentRequestRepositoryCreateIntegrationEVMCursorIsolationByNetwork(t *testing.T) {
	harness := newRepositoryIntegrationHarness(t)
	harness.resetState(t)

	sepoliaCatalog := harness.mustAssetCatalogEntry(t, "ethereum", "sepolia", "ETH")
	localCatalog := harness.mustAssetCatalogEntry(t, "ethereum", "local", "ETH")

	if sepoliaCatalog.WalletAccountID == localCatalog.WalletAccountID {
		t.Fatalf("expected distinct wallet accounts for sepolia/local, got %s", sepoliaCatalog.WalletAccountID)
	}

	_, appErr := harness.repository.Create(
		context.Background(),
		newCreatePersistenceCommand(sepoliaCatalog, "pr_sepolia_001", "idem-sepolia-001", "hash-sepolia-001", time.Now().UTC()),
		deterministicResolver,
	)
	if appErr != nil {
		t.Fatalf("expected sepolia create success, got %+v", appErr)
	}

	_, appErr = harness.repository.Create(
		context.Background(),
		newCreatePersistenceCommand(localCatalog, "pr_local_001", "idem-local-001", "hash-local-001", time.Now().UTC()),
		deterministicResolver,
	)
	if appErr != nil {
		t.Fatalf("expected local create success, got %+v", appErr)
	}

	_, appErr = harness.repository.Create(
		context.Background(),
		newCreatePersistenceCommand(localCatalog, "pr_local_002", "idem-local-002", "hash-local-002", time.Now().UTC()),
		deterministicResolver,
	)
	if appErr != nil {
		t.Fatalf("expected second local create success, got %+v", appErr)
	}

	sepoliaNextIndex := harness.mustWalletNextIndex(t, sepoliaCatalog.WalletAccountID)
	localNextIndex := harness.mustWalletNextIndex(t, localCatalog.WalletAccountID)
	if sepoliaNextIndex != 1 {
		t.Fatalf("expected sepolia next_index=1, got %d", sepoliaNextIndex)
	}
	if localNextIndex != 2 {
		t.Fatalf("expected local next_index=2, got %d", localNextIndex)
	}
}

func TestPaymentRequestRepositoryCreateIntegrationIdempotencyReplay(t *testing.T) {
	harness := newRepositoryIntegrationHarness(t)
	harness.resetState(t)

	catalog := harness.mustAssetCatalogEntry(t, "ethereum", "sepolia", "ETH")

	firstCommand := newCreatePersistenceCommand(
		catalog,
		"pr_idem_replay_001",
		"idem-replay",
		"hash-same",
		time.Now().UTC(),
	)
	firstResult, appErr := harness.repository.Create(context.Background(), firstCommand, deterministicResolver)
	if appErr != nil {
		t.Fatalf("expected first create success, got %+v", appErr)
	}
	if firstResult.Replayed {
		t.Fatalf("expected first create to be non-replayed")
	}

	secondCommand := newCreatePersistenceCommand(
		catalog,
		"pr_idem_replay_999",
		"idem-replay",
		"hash-same",
		time.Now().UTC().Add(1*time.Second),
	)
	secondResult, appErr := harness.repository.Create(context.Background(), secondCommand, deterministicResolver)
	if appErr != nil {
		t.Fatalf("expected replay success, got %+v", appErr)
	}
	if !secondResult.Replayed {
		t.Fatalf("expected replay result")
	}
	if secondResult.Resource.ID != firstResult.Resource.ID {
		t.Fatalf("expected replay resource id %s, got %s", firstResult.Resource.ID, secondResult.Resource.ID)
	}

	nextIndex := harness.mustWalletNextIndex(t, catalog.WalletAccountID)
	if nextIndex != 1 {
		t.Fatalf("expected next_index=1 after replay, got %d", nextIndex)
	}

	requestCount := harness.mustPaymentRequestCountByWallet(t, catalog.WalletAccountID)
	if requestCount != 1 {
		t.Fatalf("expected one payment request row, got %d", requestCount)
	}
}

func TestPaymentRequestRepositoryCreateIntegrationIdempotencyConflict(t *testing.T) {
	harness := newRepositoryIntegrationHarness(t)
	harness.resetState(t)

	catalog := harness.mustAssetCatalogEntry(t, "ethereum", "sepolia", "ETH")

	firstCommand := newCreatePersistenceCommand(
		catalog,
		"pr_idem_conflict_001",
		"idem-conflict",
		"hash-1",
		time.Now().UTC(),
	)
	if _, appErr := harness.repository.Create(context.Background(), firstCommand, deterministicResolver); appErr != nil {
		t.Fatalf("expected first create success, got %+v", appErr)
	}

	secondCommand := newCreatePersistenceCommand(
		catalog,
		"pr_idem_conflict_002",
		"idem-conflict",
		"hash-2",
		time.Now().UTC().Add(1*time.Second),
	)
	_, appErr := harness.repository.Create(context.Background(), secondCommand, deterministicResolver)
	if appErr == nil {
		t.Fatalf("expected idempotency conflict")
	}
	if appErr.Code != "idempotency_key_conflict" {
		t.Fatalf("expected idempotency_key_conflict, got %s", appErr.Code)
	}

	nextIndex := harness.mustWalletNextIndex(t, catalog.WalletAccountID)
	if nextIndex != 1 {
		t.Fatalf("expected next_index=1, got %d", nextIndex)
	}

	requestCount := harness.mustPaymentRequestCountByWallet(t, catalog.WalletAccountID)
	if requestCount != 1 {
		t.Fatalf("expected one payment request row, got %d", requestCount)
	}
}

func TestPaymentRequestRepositoryCreateIntegrationRollbackOnResolverFailure(t *testing.T) {
	harness := newRepositoryIntegrationHarness(t)
	harness.resetState(t)

	catalog := harness.mustAssetCatalogEntry(t, "ethereum", "sepolia", "ETH")
	command := newCreatePersistenceCommand(
		catalog,
		"pr_resolver_failure_001",
		"resolver-failure",
		"hash-failure",
		time.Now().UTC(),
	)

	_, appErr := harness.repository.Create(context.Background(), command, failingResolver)
	if appErr == nil {
		t.Fatalf("expected resolver failure")
	}
	if appErr.Code != "simulated_resolver_failure" {
		t.Fatalf("expected simulated_resolver_failure, got %s", appErr.Code)
	}

	nextIndex := harness.mustWalletNextIndex(t, catalog.WalletAccountID)
	if nextIndex != 0 {
		t.Fatalf("expected next_index=0 after rollback, got %d", nextIndex)
	}

	requestCount := harness.mustPaymentRequestCountByWallet(t, catalog.WalletAccountID)
	if requestCount != 0 {
		t.Fatalf("expected no payment request rows, got %d", requestCount)
	}

	idempotencyCount := harness.mustIdempotencyRecordCount(t, command.IdempotencyKey)
	if idempotencyCount != 0 {
		t.Fatalf("expected no idempotency record on rollback, got %d", idempotencyCount)
	}
}

func TestPaymentRequestRepositoryCreateIntegrationConcurrentSameTuple(t *testing.T) {
	harness := newRepositoryIntegrationHarness(t)
	harness.resetState(t)

	catalog := harness.mustAssetCatalogEntry(t, "ethereum", "sepolia", "ETH")

	const totalRequests = 200
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		errs  []*apperrors.AppError
		count int
	)

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			command := newCreatePersistenceCommand(
				catalog,
				fmt.Sprintf("pr_concurrent_eth_%03d", index),
				fmt.Sprintf("concurrent-eth-%03d", index),
				fmt.Sprintf("hash-concurrent-eth-%03d", index),
				time.Now().UTC(),
			)
			_, appErr := harness.repository.Create(context.Background(), command, deterministicResolver)

			mu.Lock()
			defer mu.Unlock()
			if appErr != nil {
				errs = append(errs, appErr)
				return
			}
			count++
		}(i)
	}

	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("expected no create errors, got first=%+v total=%d", errs[0], len(errs))
	}
	if count != totalRequests {
		t.Fatalf("expected %d successful requests, got %d", totalRequests, count)
	}

	walletCollisionCount := harness.mustWalletIndexCollisionCountByAsset(t, "ethereum", "sepolia", "ETH")
	if walletCollisionCount != 0 {
		t.Fatalf("expected no wallet/index collisions, got %d", walletCollisionCount)
	}

	addressCollisionCount := harness.mustAddressCollisionCountByNetwork(t, "ethereum", "sepolia")
	if addressCollisionCount != 0 {
		t.Fatalf("expected no address collisions, got %d", addressCollisionCount)
	}

	nextIndex := harness.mustWalletNextIndex(t, catalog.WalletAccountID)
	if nextIndex != int64(totalRequests) {
		t.Fatalf("expected next_index=%d, got %d", totalRequests, nextIndex)
	}
}

func TestPaymentRequestRepositoryCreateIntegrationConcurrentSharedAllocatorETHUSDT(t *testing.T) {
	harness := newRepositoryIntegrationHarness(t)
	harness.resetState(t)

	ethCatalog := harness.mustAssetCatalogEntry(t, "ethereum", "sepolia", "ETH")
	usdtCatalog := harness.mustAssetCatalogEntry(t, "ethereum", "sepolia", "USDT")

	const totalRequests = 200
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		errs  []*apperrors.AppError
		count int
	)

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			catalog := ethCatalog
			if index%2 == 1 {
				catalog = usdtCatalog
			}

			command := newCreatePersistenceCommand(
				catalog,
				fmt.Sprintf("pr_concurrent_mixed_%03d", index),
				fmt.Sprintf("concurrent-mixed-%03d", index),
				fmt.Sprintf("hash-concurrent-mixed-%03d", index),
				time.Now().UTC(),
			)
			_, appErr := harness.repository.Create(context.Background(), command, deterministicResolver)

			mu.Lock()
			defer mu.Unlock()
			if appErr != nil {
				errs = append(errs, appErr)
				return
			}
			count++
		}(i)
	}

	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("expected no create errors, got first=%+v total=%d", errs[0], len(errs))
	}
	if count != totalRequests {
		t.Fatalf("expected %d successful requests, got %d", totalRequests, count)
	}

	distinctAllocatorCount := harness.mustDistinctAllocatorCountForETHUSDT(t)
	if distinctAllocatorCount != 1 {
		t.Fatalf("expected ETH/USDT to share one allocator, got %d", distinctAllocatorCount)
	}

	walletCollisionCount := harness.mustWalletIndexCollisionCountByNetwork(t, "ethereum", "sepolia")
	if walletCollisionCount != 0 {
		t.Fatalf("expected no wallet/index collisions, got %d", walletCollisionCount)
	}

	addressCollisionCount := harness.mustAddressCollisionCountByNetwork(t, "ethereum", "sepolia")
	if addressCollisionCount != 0 {
		t.Fatalf("expected no address collisions, got %d", addressCollisionCount)
	}

	nextIndex := harness.mustWalletNextIndex(t, ethCatalog.WalletAccountID)
	if nextIndex != int64(totalRequests) {
		t.Fatalf("expected next_index=%d, got %d", totalRequests, nextIndex)
	}
}

func TestPaymentRequestRepositoryCreateIntegrationLatencyP95At20RPS(t *testing.T) {
	harness := newRepositoryIntegrationHarness(t)
	harness.resetState(t)

	catalog := harness.mustAssetCatalogEntry(t, "ethereum", "sepolia", "ETH")

	const (
		totalRequests = 40
		targetP95     = 300 * time.Millisecond
	)

	latencies := make([]time.Duration, 0, totalRequests)
	nextScheduledAt := time.Now()

	for i := 0; i < totalRequests; i++ {
		if wait := time.Until(nextScheduledAt); wait > 0 {
			time.Sleep(wait)
		}

		command := newCreatePersistenceCommand(
			catalog,
			fmt.Sprintf("pr_latency_eth_%03d", i),
			fmt.Sprintf("latency-eth-%03d", i),
			fmt.Sprintf("hash-latency-eth-%03d", i),
			time.Now().UTC(),
		)

		startedAt := time.Now()
		_, appErr := harness.repository.Create(context.Background(), command, deterministicResolver)
		if appErr != nil {
			t.Fatalf("expected create success, got %+v", appErr)
		}
		latencies = append(latencies, time.Since(startedAt))

		// 20 RPS -> one request every 50ms.
		nextScheduledAt = nextScheduledAt.Add(50 * time.Millisecond)
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	p95Index := int(math.Ceil(float64(totalRequests)*0.95)) - 1
	if p95Index < 0 {
		p95Index = 0
	}
	p95Latency := latencies[p95Index]
	if p95Latency > targetP95 {
		t.Fatalf("expected p95 <= %s at 20 RPS, got %s", targetP95, p95Latency)
	}
}

func newRepositoryIntegrationHarness(t *testing.T) *repositoryIntegrationHarness {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run integration tests")
	}

	resetDatabaseForIntegrationMigrations(t, databaseURL)

	logger := log.New(io.Discard, "", 0)
	migrationsPath := integrationMigrationsPath(t)
	bootstrapGateway := postgresqlbootstrap.NewGateway(
		databaseURL,
		"integration-target",
		migrationsPath,
		postgresqlbootstrap.ValidationRules{AllocationMode: integrationModeDevtest},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if appErr := bootstrapGateway.CheckReadiness(ctx); appErr != nil {
		t.Fatalf("expected readiness success, got %+v", appErr)
	}
	if appErr := bootstrapGateway.RunMigrations(ctx); appErr != nil {
		t.Fatalf("expected migration success, got %+v", appErr)
	}

	db := postgresqlshared.NewDatabasePool(databaseURL, logger)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return &repositoryIntegrationHarness{
		db:         db,
		repository: NewRepository(db, logger),
	}
}

func integrationMigrationsPath(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to resolve current file path")
	}

	baseDir := filepath.Dir(thisFile)
	return filepath.Clean(filepath.Join(baseDir, "..", "migrations"))
}

func newCreatePersistenceCommand(
	catalog dto.AssetCatalogEntry,
	resourceID string,
	idempotencyKey string,
	requestHash string,
	createdAt time.Time,
) dto.CreatePaymentRequestPersistenceCommand {
	return dto.CreatePaymentRequestPersistenceCommand{
		ResourceID: resourceID,
		IdempotencyScope: dto.IdempotencyScope{
			PrincipalID: integrationScopePrincipal,
			HTTPMethod:  integrationScopeMethod,
			HTTPPath:    integrationScopePath,
		},
		IdempotencyKey:       idempotencyKey,
		RequestHash:          requestHash,
		HashAlgorithm:        integrationHashAlgorithm,
		Status:               integrationStatusPending,
		Chain:                catalog.Chain,
		Network:              catalog.Network,
		Asset:                catalog.Asset,
		Metadata:             map[string]any{"test": "integration"},
		ExpiresAt:            createdAt.Add(integrationRequestTTL),
		IdempotencyExpiresAt: createdAt.Add(integrationIdempotencyTTL),
		CreatedAt:            createdAt,
		AssetCatalogSnapshot: catalog,
		AllocationMode:       integrationModeDevtest,
	}
}

func deterministicResolver(_ context.Context, input dto.ResolvePaymentAddressInput) (dto.ResolvePaymentAddressOutput, *apperrors.AppError) {
	if input.DerivationIndex < 0 {
		return dto.ResolvePaymentAddressOutput{}, apperrors.NewInternal(
			"invalid_configuration",
			"derivation index must be non-negative",
			nil,
		)
	}

	canonical := deterministicCanonicalAddress(input.Chain, input.DerivationIndex)
	return dto.ResolvePaymentAddressOutput{
		AddressCanonical: canonical,
		Address:          deterministicResponseAddress(input.Chain, canonical),
	}, nil
}

func deterministicCanonicalAddress(chain string, derivationIndex int64) string {
	switch strings.ToLower(strings.TrimSpace(chain)) {
	case "bitcoin":
		return fmt.Sprintf("bcrt1q%038x", derivationIndex+1)
	case "ethereum":
		return fmt.Sprintf("0x%040x", derivationIndex+1)
	default:
		return fmt.Sprintf("addr_%s_%d", chain, derivationIndex)
	}
}

func deterministicResponseAddress(chain string, canonical string) string {
	if !strings.EqualFold(chain, "ethereum") {
		return canonical
	}

	hexPart := strings.TrimPrefix(canonical, "0x")
	return "0x" + strings.ToUpper(hexPart)
}

func failingResolver(_ context.Context, _ dto.ResolvePaymentAddressInput) (dto.ResolvePaymentAddressOutput, *apperrors.AppError) {
	return dto.ResolvePaymentAddressOutput{}, apperrors.NewInternal(
		"simulated_resolver_failure",
		"simulated resolver failure",
		nil,
	)
}

func (h *repositoryIntegrationHarness) mustAssetCatalogEntry(t *testing.T, chain, network, asset string) dto.AssetCatalogEntry {
	t.Helper()

	const query = `
SELECT
  chain,
  network,
  asset,
  minor_unit,
  decimals,
  address_scheme,
  default_expires_in_seconds,
  chain_id,
  token_standard,
  token_contract,
  token_decimals,
  wallet_account_id
FROM app.asset_catalog
WHERE chain = $1 AND network = $2 AND asset = $3 AND enabled = TRUE
`

	var (
		entry        dto.AssetCatalogEntry
		chainID      sql.NullInt64
		tokenStd     sql.NullString
		tokenAddr    sql.NullString
		tokenDecimal sql.NullInt64
	)

	err := h.db.QueryRowContext(
		context.Background(),
		query,
		normalizeChain(chain),
		normalizeNetwork(network),
		normalizeAsset(asset),
	).Scan(
		&entry.Chain,
		&entry.Network,
		&entry.Asset,
		&entry.MinorUnit,
		&entry.Decimals,
		&entry.AddressScheme,
		&entry.DefaultExpiresInSeconds,
		&chainID,
		&tokenStd,
		&tokenAddr,
		&tokenDecimal,
		&entry.WalletAccountID,
	)
	if err != nil {
		t.Fatalf("failed to query asset catalog entry: %v", err)
	}

	if chainID.Valid {
		value := chainID.Int64
		entry.ChainID = &value
	}
	if tokenStd.Valid {
		value := tokenStd.String
		entry.TokenStandard = &value
	}
	if tokenAddr.Valid {
		value := tokenAddr.String
		entry.TokenContract = &value
	}
	if tokenDecimal.Valid {
		value := int(tokenDecimal.Int64)
		entry.TokenDecimals = &value
	}

	entry.Chain = normalizeChain(entry.Chain)
	entry.Network = normalizeNetwork(entry.Network)
	entry.Asset = normalizeAsset(entry.Asset)
	entry.AddressScheme = strings.ToLower(strings.TrimSpace(entry.AddressScheme))
	entry.WalletAccountID = strings.TrimSpace(entry.WalletAccountID)

	return entry
}

func (h *repositoryIntegrationHarness) resetState(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := h.db.ExecContext(ctx, `
DELETE FROM app.idempotency_records;
DELETE FROM app.payment_requests;
UPDATE app.wallet_accounts SET next_index = 0, updated_at = now();
`)
	if err != nil {
		t.Fatalf("failed to reset integration state: %v", err)
	}
}

func (h *repositoryIntegrationHarness) mustPaymentRequestCountByWallet(t *testing.T, walletAccountID string) int {
	t.Helper()
	return h.mustQueryInt(
		t,
		`SELECT COUNT(*) FROM app.payment_requests WHERE wallet_account_id = $1`,
		walletAccountID,
	)
}

func (h *repositoryIntegrationHarness) mustIdempotencyRecordCount(t *testing.T, idempotencyKey string) int {
	t.Helper()
	return h.mustQueryInt(
		t,
		`
SELECT COUNT(*)
FROM app.idempotency_records
WHERE scope_principal = $1
  AND scope_method = $2
  AND scope_path = $3
  AND idempotency_key = $4
`,
		integrationScopePrincipal,
		integrationScopeMethod,
		integrationScopePath,
		idempotencyKey,
	)
}

func (h *repositoryIntegrationHarness) mustWalletNextIndex(t *testing.T, walletAccountID string) int64 {
	t.Helper()

	var nextIndex int64
	err := h.db.QueryRowContext(
		context.Background(),
		`SELECT next_index FROM app.wallet_accounts WHERE id = $1`,
		walletAccountID,
	).Scan(&nextIndex)
	if err != nil {
		t.Fatalf("failed to query wallet next_index: %v", err)
	}
	return nextIndex
}

func (h *repositoryIntegrationHarness) mustAddressCanonical(t *testing.T, paymentRequestID string) string {
	t.Helper()

	var canonical string
	err := h.db.QueryRowContext(
		context.Background(),
		`SELECT address_canonical FROM app.payment_requests WHERE id = $1`,
		paymentRequestID,
	).Scan(&canonical)
	if err != nil {
		t.Fatalf("failed to query address_canonical: %v", err)
	}
	return canonical
}

func (h *repositoryIntegrationHarness) mustWalletIndexCollisionCountByAsset(t *testing.T, chain, network, asset string) int {
	t.Helper()
	return h.mustQueryInt(
		t,
		`
SELECT COUNT(*) FROM (
  SELECT wallet_account_id, derivation_index
  FROM app.payment_requests
  WHERE chain = $1 AND network = $2 AND asset = $3
  GROUP BY wallet_account_id, derivation_index
  HAVING COUNT(*) > 1
) collisions
`,
		normalizeChain(chain),
		normalizeNetwork(network),
		normalizeAsset(asset),
	)
}

func (h *repositoryIntegrationHarness) mustWalletIndexCollisionCountByNetwork(t *testing.T, chain, network string) int {
	t.Helper()
	return h.mustQueryInt(
		t,
		`
SELECT COUNT(*) FROM (
  SELECT wallet_account_id, derivation_index
  FROM app.payment_requests
  WHERE chain = $1 AND network = $2
  GROUP BY wallet_account_id, derivation_index
  HAVING COUNT(*) > 1
) collisions
`,
		normalizeChain(chain),
		normalizeNetwork(network),
	)
}

func (h *repositoryIntegrationHarness) mustAddressCollisionCountByNetwork(t *testing.T, chain, network string) int {
	t.Helper()
	return h.mustQueryInt(
		t,
		`
SELECT COUNT(*) FROM (
  SELECT chain, network, address_canonical
  FROM app.payment_requests
  WHERE chain = $1 AND network = $2
  GROUP BY chain, network, address_canonical
  HAVING COUNT(*) > 1
) collisions
`,
		normalizeChain(chain),
		normalizeNetwork(network),
	)
}

func (h *repositoryIntegrationHarness) mustDistinctAllocatorCountForETHUSDT(t *testing.T) int {
	t.Helper()
	return h.mustQueryInt(
		t,
		`
SELECT COUNT(DISTINCT wallet_account_id)
FROM app.asset_catalog
WHERE chain = 'ethereum'
  AND network = 'sepolia'
  AND asset IN ('ETH', 'USDT')
`,
	)
}

func (h *repositoryIntegrationHarness) mustQueryInt(t *testing.T, query string, args ...any) int {
	t.Helper()

	var value int
	if err := h.db.QueryRowContext(context.Background(), query, args...).Scan(&value); err != nil {
		t.Fatalf("failed to run query: %v", err)
	}
	return value
}

func normalizeChain(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeNetwork(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeAsset(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func resetDatabaseForIntegrationMigrations(t *testing.T, databaseURL string) {
	t.Helper()
	assertSafeIntegrationDatabaseURL(t, databaseURL)

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("failed to open db for migration reset: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err = db.ExecContext(ctx, `
DROP SCHEMA IF EXISTS app CASCADE;
DROP TABLE IF EXISTS schema_migrations;
`)
	if err != nil {
		t.Fatalf("failed to reset migration state: %v", err)
	}
}

func assertSafeIntegrationDatabaseURL(t *testing.T, databaseURL string) {
	t.Helper()

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("invalid TEST_DATABASE_URL: %v", err)
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	dbName := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(parsed.Path), "/"))
	hostAllowed := host == "localhost" || host == "127.0.0.1" || host == "postgres"
	dbAllowed := dbName == "chaintx" || strings.Contains(dbName, "test")

	if !hostAllowed || !dbAllowed {
		t.Fatalf("unsafe TEST_DATABASE_URL for destructive integration reset: host=%q db=%q", host, dbName)
	}
}
