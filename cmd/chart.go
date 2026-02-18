package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/aure/syntrack/internal/db"
	"github.com/spf13/cobra"
)

var chartDays int
var chartType string

var chartCmd = &cobra.Command{
	Use:   "chart",
	Short: "Display ASCII usage charts",
	Long: `Display ASCII charts for usage visualization.

Chart types:
  usage     - Used vs leftover over time (default)
  daily     - Daily consumption bar chart
  weekly    - Weekly consumption bar chart

Examples:
  syntrack chart
  syntrack chart --type daily
  syntrack chart --days 14 --type usage`,
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.New(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}

		switch chartType {
		case "usage":
			return printUsageChart(database, chartDays)
		case "daily":
			return printDailyChart(database, chartDays)
		case "weekly":
			return printWeeklyChart(database)
		default:
			return fmt.Errorf("unknown chart type: %s (valid: usage, daily, weekly)", chartType)
		}
	},
}

func printUsageChart(database *db.DB, days int) error {
	since := time.Now().AddDate(0, 0, -days)
	snapshots, err := database.GetSnapshots(since)
	if err != nil {
		return err
	}

	if len(snapshots) < 2 {
		fmt.Println("Need at least 2 snapshots. Run 'syntrack collect' more often.")
		return nil
	}

	printASCIIChart(snapshots)
	return nil
}

func printDailyChart(database *db.DB, days int) error {
	daily, err := database.GetDailyUsage(days)
	if err != nil {
		return err
	}

	if len(daily) == 0 {
		fmt.Println("No data available.")
		return nil
	}

	maxConsumed := 0
	for _, d := range daily {
		if d.RequestsConsumed > maxConsumed {
			maxConsumed = d.RequestsConsumed
		}
	}

	if maxConsumed == 0 {
		maxConsumed = 1
	}

	barWidth := 30

	fmt.Println()
	fmt.Println("  Daily Requests Consumed")
	fmt.Println("  " + strings.Repeat("─", 50))

	for _, d := range daily {
		barLen := int(float64(d.RequestsConsumed) / float64(maxConsumed) * float64(barWidth))
		bar := strings.Repeat("█", barLen) + strings.Repeat("░", barWidth-barLen)
		fmt.Printf("  %s │%s│ %d\n", d.Day, bar, d.RequestsConsumed)
	}

	fmt.Println()
	return nil
}

func printWeeklyChart(database *db.DB) error {
	weekly, err := database.GetWeeklyUsage(8)
	if err != nil {
		return err
	}

	if len(weekly) == 0 {
		fmt.Println("No data available.")
		return nil
	}

	maxConsumed := 0
	for _, w := range weekly {
		if w.RequestsConsumed > maxConsumed {
			maxConsumed = w.RequestsConsumed
		}
	}

	if maxConsumed == 0 {
		maxConsumed = 1
	}

	barWidth := 30

	fmt.Println()
	fmt.Println("  Weekly Requests Consumed")
	fmt.Println("  " + strings.Repeat("─", 50))

	for _, w := range weekly {
		barLen := int(float64(w.RequestsConsumed) / float64(maxConsumed) * float64(barWidth))
		bar := strings.Repeat("█", barLen) + strings.Repeat("░", barWidth-barLen)
		fmt.Printf("  %s │%s│ %d\n", w.Week, bar, w.RequestsConsumed)
	}

	fmt.Println()
	return nil
}

func init() {
	chartCmd.Flags().IntVarP(&chartDays, "days", "d", 7, "Number of days to display")
	chartCmd.Flags().StringVarP(&chartType, "type", "t", "usage", "Chart type (usage, daily, weekly)")
	rootCmd.AddCommand(chartCmd)
}
