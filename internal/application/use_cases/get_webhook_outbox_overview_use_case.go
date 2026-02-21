package use_cases

import (
	"context"
	"time"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type getWebhookOutboxOverviewUseCase struct {
	readModel portsout.WebhookOutboxReadModel
}

func NewGetWebhookOutboxOverviewUseCase(readModel portsout.WebhookOutboxReadModel) portsin.GetWebhookOutboxOverviewUseCase {
	return &getWebhookOutboxOverviewUseCase{readModel: readModel}
}

func (u *getWebhookOutboxOverviewUseCase) Execute(
	ctx context.Context,
	query dto.GetWebhookOutboxOverviewQuery,
) (dto.WebhookOutboxOverview, *apperrors.AppError) {
	if u.readModel == nil {
		return dto.WebhookOutboxOverview{}, apperrors.NewInternal(
			"webhook_outbox_read_model_missing",
			"webhook outbox read model is required",
			nil,
		)
	}

	now := query.Now.UTC()
	if query.Now.IsZero() {
		now = time.Now().UTC()
	}

	return u.readModel.GetOverview(ctx, now)
}
