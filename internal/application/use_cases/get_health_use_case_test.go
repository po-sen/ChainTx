package use_cases

import (
	"context"
	"testing"

	"chaintx/internal/application/dto"
)

func TestGetHealthUseCase_Execute(t *testing.T) {
	useCase := NewGetHealthUseCase()

	output, appErr := useCase.Execute(context.Background(), dto.GetHealthCommand{})
	if appErr != nil {
		t.Fatalf("expected no error, got %v", appErr)
	}

	if output.Status != "ok" {
		t.Fatalf("expected status to be ok, got %q", output.Status)
	}
}
