package models

type OverviewSLA struct {
	FirstResponseMetCount      int     `json:"first_response_met_count" db:"first_response_met_count"`
	FirstResponseBreachedCount int     `json:"first_response_breached_count" db:"first_response_breached_count"`
	AvgFirstResponseTimeSec    float64 `json:"avg_first_response_time_sec" db:"avg_first_response_time_sec"`
	NextResponseMetCount       int     `json:"next_response_met_count" db:"next_response_met_count"`
	NextResponseBreachedCount  int     `json:"next_response_breached_count" db:"next_response_breached_count"`
	AvgNextResponseTimeSec     float64 `json:"avg_next_response_time_sec" db:"avg_next_response_time_sec"`
	ResolutionMetCount         int     `json:"resolution_met_count" db:"resolution_met_count"`
	ResolutionBreachedCount    int     `json:"resolution_breached_count" db:"resolution_breached_count"`
	AvgResolutionTimeSec       float64 `json:"avg_resolution_time_sec" db:"avg_resolution_time_sec"`
}
