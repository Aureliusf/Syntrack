package cmd

import (
	"fmt"
	"time"

	"github.com/aure/syntrack/internal/db"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show usage statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.New(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}

		latest, err := database.GetLatestSnapshot()
		if err != nil {
			return fmt.Errorf("getting latest snapshot: %w", err)
		}

		if latest == nil {
			fmt.Println("No data available. Run 'syntrack collect' first.")
			return nil
		}

		burnRate, err := database.GetBurnRate(24)
		if err != nil {
			return fmt.Errorf("calculating burn rate: %w", err)
		}

		fmt.Println("═══════════════════════════════════════")
		fmt.Println("         USAGE STATISTICS")
		fmt.Println("═══════════════════════════════════════")

		fmt.Println("\n📊 Current Status")
		fmt.Println("─────────────────────")
		pct := float64(latest.RequestsUsed) / float64(latest.SubscriptionLimit) * 100
		fmt.Printf("  Used:      %d / %d (%.1f%%)\n", latest.RequestsUsed, latest.SubscriptionLimit, pct)
		fmt.Printf("  Leftover:  %d\n", latest.Leftover)

		fmt.Println("\n🔥 Burn Rate (24h)")
		fmt.Println("─────────────────────")
		if burnRate > 0 {
			hoursLeft := float64(latest.Leftover) / burnRate
			fmt.Printf("  Rate:      %.2f requests/hour\n", burnRate)
			fmt.Printf("  Est. left: %.1f hours (%.1f days)\n", hoursLeft, hoursLeft/24)
		} else {
			fmt.Println("  Not enough data to calculate")
		}

		fmt.Println("\n📅 Daily Usage")
		fmt.Println("─────────────────────")
		daily, err := database.GetDailyUsage(7)
		if err != nil {
			return fmt.Errorf("getting daily usage: %w", err)
		}
		if len(daily) == 0 {
			fmt.Println("  No data yet")
		} else {
			fmt.Printf("%-12s %10s %10s\n", "Day", "Consumed", "Avg Left")
			for _, d := range daily {
				fmt.Printf("%-12s %10d %10.0f\n", d.Day, d.RequestsConsumed, d.AvgLeftover)
			}
		}

		fmt.Println("\n📆 Weekly Usage")
		fmt.Println("─────────────────────")
		weekly, err := database.GetWeeklyUsage(4)
		if err != nil {
			return fmt.Errorf("getting weekly usage: %w", err)
		}
		if len(weekly) == 0 {
			fmt.Println("  No data yet")
		} else {
			fmt.Printf("%-12s %10s %10s\n", "Week", "Consumed", "Avg Left")
			for _, w := range weekly {
				fmt.Printf("%-12s %10d %10.0f\n", w.Week, w.RequestsConsumed, w.AvgLeftover)
			}
		}

		fmt.Println("\n📈 Overall")
		fmt.Println("─────────────────────")
		snapshots, err := database.GetSnapshots(time.Time{})
		if err != nil {
			return fmt.Errorf("getting all snapshots: %w", err)
		}
		if len(snapshots) > 0 {
			fmt.Printf("  Total snapshots: %d\n", len(snapshots))
			fmt.Printf("  First snapshot:  %s\n", snapshots[0].CollectedAt.Format("2006-01-02 15:04"))
			fmt.Printf("  Latest snapshot: %s\n", snapshots[len(snapshots)-1].CollectedAt.Format("2006-01-02 15:04"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
