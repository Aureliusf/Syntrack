package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/aure/syntrack/internal/api"
	"github.com/aure/syntrack/internal/db"
	"github.com/spf13/cobra"
)

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect usage data from Synthetic API",
	RunE: func(cmd *cobra.Command, args []string) error {
		if apiKey == "" {
			return fmt.Errorf("SYNTHETIC_API_KEY not set")
		}

		database, err := db.New(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}

		client := api.NewClient(apiKey)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		quota, err := client.GetQuotas(ctx)
		if err != nil {
			return fmt.Errorf("fetching quotas: %w", err)
		}

		var renewsAt *time.Time
		if !quota.Subscription.RenewsAt.IsZero() {
			renewsAt = &quota.Subscription.RenewsAt
		}

		if err := database.InsertSnapshot(quota.Subscription.Limit, quota.Subscription.Requests, renewsAt); err != nil {
			return fmt.Errorf("inserting snapshot: %w", err)
		}

		leftover := quota.Subscription.Limit - quota.Subscription.Requests
		fmt.Printf("Collected: %d/%d used (%d leftover)\n", quota.Subscription.Requests, quota.Subscription.Limit, leftover)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(collectCmd)
}
