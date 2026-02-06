package dto

type GetHealthCommand struct{}

type HealthOutput struct {
	Status string `json:"status"`
}
