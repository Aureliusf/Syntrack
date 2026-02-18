package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/aure/syntrack/internal/db"
	"github.com/spf13/cobra"
)

var historyDays int
var historyWeeks int
var historyChart bool

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

		if historyChart {
			printASCIIChart(snapshots)
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

func printASCIIChart(snapshots []db.UsageSnapshot) {
	if len(snapshots) < 2 {
		fmt.Println("Need at least 2 data points for a chart")
		return
	}

	width := 60
	height := 15

	maxVal := float64(snapshots[0].SubscriptionLimit)
	for _, s := range snapshots {
		if float64(s.RequestsUsed) > maxVal {
			maxVal = float64(s.RequestsUsed)
		}
	}

	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = []rune(strings.Repeat(" ", width))
	}

	for i, s := range snapshots {
		x := int(float64(i) / float64(len(snapshots)-1) * float64(width-1))
		if x >= width {
			x = width - 1
		}

		yUsed := int((1 - float64(s.RequestsUsed)/maxVal) * float64(height-1))
		if yUsed < 0 {
			yUsed = 0
		}
		if yUsed >= height {
			yUsed = height - 1
		}

		yLeft := int((1 - float64(s.Leftover)/maxVal) * float64(height-1))
		if yLeft < 0 {
			yLeft = 0
		}
		if yLeft >= height {
			yLeft = height - 1
		}

		if grid[yUsed][x] == ' ' {
			grid[yUsed][x] = '#'
		}
		if grid[yLeft][x] == ' ' {
			grid[yLeft][x] = '.'
		}
	}

	fmt.Println()
	fmt.Printf("     Usage Chart (last %d data points)\n", len(snapshots))
	fmt.Println("     " + strings.Repeat("─", width))
	for y := 0; y < height; y++ {
		label := "    "
		if y == 0 {
			label = fmt.Sprintf("%4d", int(maxVal))
		} else if y == height-1 {
			label = "   0"
		}
		fmt.Printf("%s │", label)
		for x := 0; x < width; x++ {
			fmt.Printf("%c", grid[y][x])
		}
		fmt.Println()
	}
	fmt.Printf("     └" + strings.Repeat("─", width))
	fmt.Println()

	labels := 5
	step := len(snapshots) / labels
	if step < 1 {
		step = 1
	}
	fmt.Printf("       ")
	for i := 0; i < len(snapshots); i += step {
		pos := int(float64(i) / float64(len(snapshots)-1) * float64(width-1))
		if i == 0 {
			fmt.Printf("%s", snapshots[i].CollectedAt.Format("01/02"))
		} else if pos < width-5 {
			fmt.Printf("%*s%s", pos-5, "", snapshots[i].CollectedAt.Format("01/02"))
		}
	}
	fmt.Println()

	fmt.Println()
	fmt.Println("Legend: # = Used  . = Leftover")
	fmt.Printf("Data range: %s to %s\n",
		snapshots[0].CollectedAt.Format("2006-01-02 15:04"),
		snapshots[len(snapshots)-1].CollectedAt.Format("2006-01-02 15:04"))
}

func init() {
	historyCmd.Flags().IntVarP(&historyDays, "days", "d", 7, "Number of days to show")
	historyCmd.Flags().BoolVarP(&historyChart, "chart", "c", false, "Show ASCII chart instead of table")
	rootCmd.AddCommand(historyCmd)
}
