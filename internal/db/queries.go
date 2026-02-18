package db

import (
	"database/sql"
	"time"
)

type UsageSnapshot struct {
	ID                int64
	CollectedAt       time.Time
	SubscriptionLimit int
	RequestsUsed      int
	Leftover          int
	RenewsAt          *time.Time
}

type DailyUsage struct {
	Day              string
	RequestsConsumed int
	MinLeftover      float64
	MaxLeftover      float64
	AvgLeftover      float64
	Snapshots        int
}

type WeeklyUsage struct {
	Week             string
	RequestsConsumed int
	MinLeftover      float64
	MaxLeftover      float64
	AvgLeftover      float64
	Snapshots        int
}

func (db *DB) InsertSnapshot(limit, requests int, renewsAt *time.Time) error {
	_, err := db.Exec(
		`INSERT INTO usage_snapshots (subscription_limit, requests_used, renews_at) VALUES (?, ?, ?)`,
		limit, requests, renewsAt,
	)
	return err
}

func (db *DB) GetLatestSnapshot() (*UsageSnapshot, error) {
	row := db.QueryRow(`SELECT id, collected_at, subscription_limit, requests_used, leftover, renews_at FROM usage_snapshots ORDER BY collected_at DESC LIMIT 1`)

	var s UsageSnapshot
	var renewsAt sql.NullTime
	err := row.Scan(&s.ID, &s.CollectedAt, &s.SubscriptionLimit, &s.RequestsUsed, &s.Leftover, &renewsAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if renewsAt.Valid {
		s.RenewsAt = &renewsAt.Time
	}
	return &s, nil
}

func (db *DB) GetSnapshots(since time.Time) ([]UsageSnapshot, error) {
	rows, err := db.Query(`SELECT id, collected_at, subscription_limit, requests_used, leftover, renews_at FROM usage_snapshots WHERE collected_at >= ? ORDER BY collected_at ASC`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []UsageSnapshot
	for rows.Next() {
		var s UsageSnapshot
		var renewsAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.CollectedAt, &s.SubscriptionLimit, &s.RequestsUsed, &s.Leftover, &renewsAt); err != nil {
			return nil, err
		}
		if renewsAt.Valid {
			s.RenewsAt = &renewsAt.Time
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, rows.Err()
}

func (db *DB) GetDailyUsage(days int) ([]DailyUsage, error) {
	rows, err := db.Query(`SELECT day, requests_consumed, min_leftover, max_leftover, avg_leftover, snapshots FROM daily_usage LIMIT ?`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DailyUsage
	for rows.Next() {
		var d DailyUsage
		if err := rows.Scan(&d.Day, &d.RequestsConsumed, &d.MinLeftover, &d.MaxLeftover, &d.AvgLeftover, &d.Snapshots); err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, rows.Err()
}

func (db *DB) GetWeeklyUsage(weeks int) ([]WeeklyUsage, error) {
	rows, err := db.Query(`SELECT week, requests_consumed, min_leftover, max_leftover, avg_leftover, snapshots FROM weekly_usage LIMIT ?`, weeks)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []WeeklyUsage
	for rows.Next() {
		var w WeeklyUsage
		if err := rows.Scan(&w.Week, &w.RequestsConsumed, &w.MinLeftover, &w.MaxLeftover, &w.AvgLeftover, &w.Snapshots); err != nil {
			return nil, err
		}
		results = append(results, w)
	}
	return results, rows.Err()
}

func (db *DB) GetBurnRate(hours int) (float64, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	snapshots, err := db.GetSnapshots(since)
	if err != nil {
		return 0, err
	}
	if len(snapshots) < 2 {
		return 0, nil
	}

	first := snapshots[0]
	last := snapshots[len(snapshots)-1]
	requestsDiff := last.RequestsUsed - first.RequestsUsed
	timeDiff := last.CollectedAt.Sub(first.CollectedAt).Hours()

	if timeDiff <= 0 {
		return 0, nil
	}
	return float64(requestsDiff) / timeDiff, nil
}
