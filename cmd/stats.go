package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/aure/syntrack/internal/db"
	"github.com/spf13/cobra"
)

var statsChart bool

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
		printMiniBar(latest.RequestsUsed, latest.SubscriptionLimit, 30)

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
		} else if statsChart {
			maxConsumed := 1
			for _, d := range daily {
				if d.RequestsConsumed > maxConsumed {
					maxConsumed = d.RequestsConsumed
				}
			}
			for _, d := range daily {
				barLen := int(float64(d.RequestsConsumed) / float64(maxConsumed) * 20)
				bar := strings.Repeat("█", barLen) + strings.Repeat("░", 20-barLen)
				fmt.Printf("  %s %s %d\n", d.Day, bar, d.RequestsConsumed)
			}
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
		} else if statsChart {
			maxConsumed := 1
			for _, w := range weekly {
				if w.RequestsConsumed > maxConsumed {
					maxConsumed = w.RequestsConsumed
				}
			}
			for _, w := range weekly {
				barLen := int(float64(w.RequestsConsumed) / float64(maxConsumed) * 20)
				bar := strings.Repeat("█", barLen) + strings.Repeat("░", 20-barLen)
				fmt.Printf("  %s %s %d\n", w.Week, bar, w.RequestsConsumed)
			}
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

			if len(snapshots) > 1 {
				totalConsumed := snapshots[len(snapshots)-1].RequestsUsed - snapshots[0].RequestsUsed
				days := snapshots[len(snapshots)-1].CollectedAt.Sub(snapshots[0].CollectedAt).Hours() / 24
				if days > 0 {
					fmt.Printf("  Avg daily:       %.1f requests/day\n", float64(totalConsumed)/days)
				}
			}
		}

		if statsChart && len(snapshots) > 1 {
			fmt.Println("\n📉 Usage Trend (last 7 days)")
			fmt.Println("─────────────────────")
			since := time.Now().AddDate(0, 0, -7)
			recent, _ := database.GetSnapshots(since)
			if len(recent) > 1 {
				printSparkline(recent)
			}
		}

		return nil
	},
}

func printMiniBar(used, total, width int) {
	pct := float64(used) / float64(total)
	filled := int(pct * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	fmt.Printf("  [%s] %.0f%%\n", bar, pct*100)
}

func printSparkline(snapshots []db.UsageSnapshot) {
	if len(snapshots) < 2 {
		return
	}

	width := 40

	maxVal := float64(snapshots[0].SubscriptionLimit)
	for _, s := range snapshots {
		if float64(s.RequestsUsed) > maxVal {
			maxVal = float64(s.RequestsUsed)
		}
	}

	chars := []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var sparkline strings.Builder
	for i := 0; i < len(snapshots); i++ {
		step := float64(len(snapshots)-1) / float64(width-1)
		idx := int(float64(i) / step)
		if idx >= len(snapshots) {
			idx = len(snapshots) - 1
		}

		pct := float64(snapshots[idx].RequestsUsed) / maxVal
		charIdx := int(pct * float64(len(chars)-1))
		if charIdx >= len(chars) {
			charIdx = len(chars) - 1
		}
		if charIdx < 0 {
			charIdx = 0
		}
		sparkline.WriteRune(chars[charIdx])
	}

	fmt.Printf("  Used:  %s\n", sparkline.String())

	sparkline.Reset()
	for i := 0; i < len(snapshots); i++ {
		step := float64(len(snapshots)-1) / float64(width-1)
		idx := int(float64(i) / step)
		if idx >= len(snapshots) {
			idx = len(snapshots) - 1
		}

		pct := float64(snapshots[idx].Leftover) / maxVal
		charIdx := int(pct * float64(len(chars)-1))
		if charIdx >= len(chars) {
			charIdx = len(chars) - 1
		}
		if charIdx < 0 {
			charIdx = 0
		}
		sparkline.WriteRune(chars[charIdx])
	}

	fmt.Printf("  Left:  %s\n", sparkline.String())
}

func init() {
	statsCmd.Flags().BoolVarP(&statsChart, "chart", "c", false, "Show ASCII charts inline")
	rootCmd.AddCommand(statsCmd)
}
