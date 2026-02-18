package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aure/syntrack/internal/db"
	"github.com/spf13/cobra"
)

var queryOutput string

var queryCmd = &cobra.Command{
	Use:   "query [type]",
	Short: "Query usage data in JSON format (for agents/scripts)",
	Long: `Query usage data in structured JSON format.

Types:
  current    - Current quota status
  today      - Today's usage summary
  yesterday  - Yesterday's usage summary
  week       - This week's usage
  burn-rate  - Current burn rate and predictions
  history    - Recent snapshots (use --days flag)
  daily      - Daily breakdown (use --days flag)
  weekly     - Weekly breakdown (use --weeks flag)

Examples:
  syntrack query current
  syntrack query today
  syntrack query history --days 3
  syntrack query daily --days 7`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.New(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}

		queryType := args[0]
		var result any

		switch queryType {
		case "current":
			result, err = queryCurrent(database)
		case "today":
			result, err = queryDay(database, 0)
		case "yesterday":
			result, err = queryDay(database, -1)
		case "week":
			result, err = queryWeek(database)
		case "burn-rate":
			result, err = queryBurnRate(database)
		case "history":
			result, err = queryHistory(database, historyDays)
		case "daily":
			result, err = queryDaily(database, historyDays)
		case "weekly":
			result, err = queryWeekly(database, 4)
		default:
			return fmt.Errorf("unknown query type: %s (valid: current, today, yesterday, week, burn-rate, history, daily, weekly)", queryType)
		}

		if err != nil {
			return err
		}

		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}

		fmt.Println(string(output))
		return nil
	},
}

type CurrentStatus struct {
	Timestamp       string  `json:"timestamp"`
	Limit           int     `json:"limit"`
	Used            int     `json:"used"`
	Leftover        int     `json:"leftover"`
	UsagePercent    float64 `json:"usage_percent"`
	RenewsAt        *string `json:"renews_at,omitempty"`
	TimeUntilRenew  string  `json:"time_until_renew,omitempty"`
}

func queryCurrent(database *db.DB) (any, error) {
	snapshot, err := database.GetLatestSnapshot()
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return CurrentStatus{Timestamp: time.Now().Format(time.RFC3339)}, nil
	}

	status := CurrentStatus{
		Timestamp:    snapshot.CollectedAt.Format(time.RFC3339),
		Limit:        snapshot.SubscriptionLimit,
		Used:         snapshot.RequestsUsed,
		Leftover:     snapshot.Leftover,
		UsagePercent: float64(snapshot.RequestsUsed) / float64(snapshot.SubscriptionLimit) * 100,
	}

	if snapshot.RenewsAt != nil {
		r := snapshot.RenewsAt.Format(time.RFC3339)
		status.RenewsAt = &r
		status.TimeUntilRenew = time.Until(*snapshot.RenewsAt).Round(time.Minute).String()
	}

	return status, nil
}

type DaySummary struct {
	Date            string  `json:"date"`
	RequestsUsed    int     `json:"requests_used"`
	RequestsLimit   int     `json:"requests_limit"`
	Leftover        int     `json:"leftover"`
	ConsumedToday   int     `json:"consumed_today"`
	Snapshots       int     `json:"snapshots"`
	UsagePercent    float64 `json:"usage_percent"`
}

func queryDay(database *db.DB, dayOffset int) (any, error) {
	date := time.Now().AddDate(0, 0, dayOffset).Format("2006-01-02")
	start, _ := time.Parse("2006-01-02", date)
	end := start.Add(24 * time.Hour)

	snapshots, err := database.GetSnapshots(start)
	if err != nil {
		return nil, err
	}

	var filtered []db.UsageSnapshot
	for _, s := range snapshots {
		if s.CollectedAt.Before(end) {
			filtered = append(filtered, s)
		}
	}

	if len(filtered) == 0 {
		return DaySummary{Date: date}, nil
	}

	first := filtered[0]
	last := filtered[len(filtered)-1]
	consumed := last.RequestsUsed - first.RequestsUsed
	if consumed < 0 {
		consumed = 0
	}

	return DaySummary{
		Date:          date,
		RequestsUsed:  last.RequestsUsed,
		RequestsLimit: last.SubscriptionLimit,
		Leftover:      last.Leftover,
		ConsumedToday: consumed,
		Snapshots:     len(filtered),
		UsagePercent:  float64(last.RequestsUsed) / float64(last.SubscriptionLimit) * 100,
	}, nil
}

type WeekSummary struct {
	WeekStart       string  `json:"week_start"`
	WeekEnd         string  `json:"week_end"`
	RequestsUsed    int     `json:"requests_used"`
	RequestsLimit   int     `json:"requests_limit"`
	Leftover        int     `json:"leftover"`
	ConsumedThisWeek int    `json:"consumed_this_week"`
	Snapshots       int     `json:"snapshots"`
	UsagePercent    float64 `json:"usage_percent"`
}

func queryWeek(database *db.DB) (any, error) {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -weekday+1).Truncate(24 * time.Hour)
	weekEnd := weekStart.AddDate(0, 0, 7)

	snapshots, err := database.GetSnapshots(weekStart)
	if err != nil {
		return nil, err
	}

	var filtered []db.UsageSnapshot
	for _, s := range snapshots {
		if s.CollectedAt.Before(weekEnd) {
			filtered = append(filtered, s)
		}
	}

	if len(filtered) == 0 {
		return WeekSummary{
			WeekStart: weekStart.Format("2006-01-02"),
			WeekEnd:   weekEnd.Format("2006-01-02"),
		}, nil
	}

	first := filtered[0]
	last := filtered[len(filtered)-1]
	consumed := last.RequestsUsed - first.RequestsUsed
	if consumed < 0 {
		consumed = 0
	}

	return WeekSummary{
		WeekStart:        weekStart.Format("2006-01-02"),
		WeekEnd:          weekEnd.Format("2006-01-02"),
		RequestsUsed:     last.RequestsUsed,
		RequestsLimit:    last.SubscriptionLimit,
		Leftover:         last.Leftover,
		ConsumedThisWeek: consumed,
		Snapshots:        len(filtered),
		UsagePercent:     float64(last.RequestsUsed) / float64(last.SubscriptionLimit) * 100,
	}, nil
}

type BurnRateResult struct {
	CalculatedAt      string  `json:"calculated_at"`
	RatePerHour       float64 `json:"rate_per_hour"`
	RatePerDay        float64 `json:"rate_per_day"`
	CurrentLeftover   int     `json:"current_leftover"`
	HoursUntilEmpty   float64 `json:"hours_until_empty"`
	DaysUntilEmpty    float64 `json:"days_until_empty"`
	EstimatedEmptyAt  string  `json:"estimated_empty_at,omitempty"`
	DataPoints        int     `json:"data_points"`
	PeriodHours       int     `json:"period_hours"`
}

func queryBurnRate(database *db.DB) (any, error) {
	burnRate, err := database.GetBurnRate(24)
	if err != nil {
		return nil, err
	}

	latest, err := database.GetLatestSnapshot()
	if err != nil || latest == nil {
		return BurnRateResult{CalculatedAt: time.Now().Format(time.RFC3339)}, nil
	}

	since := time.Now().Add(-24 * time.Hour)
	snapshots, _ := database.GetSnapshots(since)

	result := BurnRateResult{
		CalculatedAt:    time.Now().Format(time.RFC3339),
		RatePerHour:     burnRate,
		RatePerDay:      burnRate * 24,
		CurrentLeftover: latest.Leftover,
		DataPoints:      len(snapshots),
		PeriodHours:     24,
	}

	if burnRate > 0 {
		result.HoursUntilEmpty = float64(latest.Leftover) / burnRate
		result.DaysUntilEmpty = result.HoursUntilEmpty / 24
		emptyAt := time.Now().Add(time.Duration(result.HoursUntilEmpty) * time.Hour)
		result.EstimatedEmptyAt = emptyAt.Format(time.RFC3339)
	}

	return result, nil
}

func queryHistory(database *db.DB, days int) (any, error) {
	since := time.Now().AddDate(0, 0, -days)
	snapshots, err := database.GetSnapshots(since)
	if err != nil {
		return nil, err
	}

	type HistoryEntry struct {
		Timestamp    string  `json:"timestamp"`
		Limit        int     `json:"limit"`
		Used         int     `json:"used"`
		Leftover     int     `json:"leftover"`
		UsagePercent float64 `json:"usage_percent"`
	}

	var history []HistoryEntry
	for _, s := range snapshots {
		history = append(history, HistoryEntry{
			Timestamp:    s.CollectedAt.Format(time.RFC3339),
			Limit:        s.SubscriptionLimit,
			Used:         s.RequestsUsed,
			Leftover:     s.Leftover,
			UsagePercent: float64(s.RequestsUsed) / float64(s.SubscriptionLimit) * 100,
		})
	}

	return history, nil
}

func queryDaily(database *db.DB, days int) (any, error) {
	return database.GetDailyUsage(days)
}

func queryWeekly(database *db.DB, weeks int) (any, error) {
	return database.GetWeeklyUsage(weeks)
}

func init() {
	queryCmd.Flags().StringVarP(&queryOutput, "output", "o", "json", "Output format (json)")
	queryCmd.Flags().IntVarP(&historyDays, "days", "d", 7, "Number of days for history/daily queries")
	queryCmd.Flags().IntVarP(&historyWeeks, "weeks", "w", 4, "Number of weeks for weekly queries")
	rootCmd.AddCommand(queryCmd)
}
