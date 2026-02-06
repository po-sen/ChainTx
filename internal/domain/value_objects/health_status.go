package valueobjects

type HealthStatus string

const (
	HealthStatusOK HealthStatus = "ok"
)

func NewHealthyStatus() HealthStatus {
	return HealthStatusOK
}

func (h HealthStatus) String() string {
	return string(h)
}
