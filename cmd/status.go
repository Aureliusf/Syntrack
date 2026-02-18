package cmd

import (
	"fmt"

	"github.com/aure/syntrack/internal/db"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current usage status",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.New(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}

		snapshot, err := database.GetLatestSnapshot()
		if err != nil {
			return fmt.Errorf("getting latest snapshot: %w", err)
		}

		if snapshot == nil {
			fmt.Println("No data collected yet. Run 'syntrack collect' first.")
			return nil
		}

		pct := float64(snapshot.RequestsUsed) / float64(snapshot.SubscriptionLimit) * 100

		fmt.Printf("Current Status (as of %s)\n", snapshot.CollectedAt.Format("2006-01-02 15:04"))
		fmt.Println("─────────────────────────────")
		fmt.Printf("  Limit:     %d\n", snapshot.SubscriptionLimit)
		fmt.Printf("  Used:      %d\n", snapshot.RequestsUsed)
		fmt.Printf("  Leftover:  %d\n", snapshot.Leftover)
		fmt.Printf("  Usage:     %.1f%%\n", pct)

		if snapshot.RenewsAt != nil {
			fmt.Printf("  Renews:    %s\n", snapshot.RenewsAt.Format("2006-01-02 15:04"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
