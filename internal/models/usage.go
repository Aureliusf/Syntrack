package models

import "time"

type Snapshot struct {
	CollectedAt       time.Time
	SubscriptionLimit int
	RequestsUsed      int
	Leftover          int
	RenewsAt          *time.Time
}

type DailyStats struct {
	Day              string
	RequestsConsumed int
	MinLeftover      float64
	MaxLeftover      float64
	AvgLeftover      float64
}

type WeeklyStats struct {
	Week             string
	RequestsConsumed int
	MinLeftover      float64
	MaxLeftover      float64
	AvgLeftover      float64
}

type BurnRateStats struct {
	RequestsPerHour float64
	HoursUntilEmpty float64
	CurrentLeftover int
}

type OverallStats struct {
	TotalSnapshots int
	TotalRequests  int64
	AvgDailyUsage  float64
	FirstSnapshot  time.Time
	LatestSnapshot time.Time
}
