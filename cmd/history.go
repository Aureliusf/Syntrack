package cmd

import (
	"fmt"
	"time"

	"github.com/aure/syntrack/internal/db"
	"github.com/spf13/cobra"
)

var historyDays int

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show usage history",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.New(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}

		since := time.Now().AddDate(0, 0, -historyDays)
		snapshots, err := database.GetSnapshots(since)
		if err != nil {
			return fmt.Errorf("getting snapshots: %w", err)
		}

		if len(snapshots) == 0 {
			fmt.Println("No data available. Run 'syntrack collect' first.")
			return nil
		}

		fmt.Printf("Usage History (last %d days)\n", historyDays)
		fmt.Println("─────────────────────────────────────────────────────────────────")
		fmt.Printf("%-20s %8s %8s %8s %8s\n", "Time", "Limit", "Used", "Left", "%")
		fmt.Println("─────────────────────────────────────────────────────────────────")

		for i := len(snapshots) - 1; i >= 0; i-- {
			s := snapshots[i]
			pct := float64(s.RequestsUsed) / float64(s.SubscriptionLimit) * 100
			fmt.Printf("%-20s %8d %8d %8d %7.1f%%\n",
				s.CollectedAt.Format("2006-01-02 15:04"),
				s.SubscriptionLimit,
				s.RequestsUsed,
				s.Leftover,
				pct,
			)
		}

		return nil
	},
}

func init() {
	historyCmd.Flags().IntVarP(&historyDays, "days", "d", 7, "Number of days to show")
	rootCmd.AddCommand(historyCmd)
}
