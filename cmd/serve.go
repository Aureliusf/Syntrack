package cmd

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/aure/syntrack/internal/config"
	"github.com/aure/syntrack/internal/db"
	"github.com/spf13/cobra"
)

var servePort int
var requireAuth bool
var authTokens []string

func tokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if not required or no tokens configured
		if !requireAuth || len(authTokens) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// Allow localhost requests without token
		host := r.Host
		if strings.HasPrefix(host, "localhost:") || strings.HasPrefix(host, "127.0.0.1:") || strings.HasPrefix(host, "[::1]:") {
			next.ServeHTTP(w, r)
			return
		}

		// Check X-Auth-Token header
		token := r.Header.Get("X-Auth-Token")
		if token == "" {
			http.Error(w, "Unauthorized: X-Auth-Token header required", http.StatusUnauthorized)
			return
		}

		// Validate token
		valid := false
		for _, t := range authTokens {
			if t == token {
				valid = true
				break
			}
		}

		if !valid {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web dashboard server",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.New(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}

		mux := http.NewServeMux()

		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

		partials := template.Must(template.New("").ParseGlob("web/templates/partials/*.html"))

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			tmpl, err := partials.Clone()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			tmpl, err = tmpl.ParseFiles("web/templates/layout.html", "web/templates/index.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			if err := tmpl.ExecuteTemplate(w, "layout.html", nil); err != nil {
				fmt.Printf("Template error: %v\n", err)
			}
		})
		mux.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) {
			tmpl, err := partials.Clone()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			tmpl, err = tmpl.ParseFiles("web/templates/layout.html", "web/templates/history.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			if err := tmpl.ExecuteTemplate(w, "layout.html", nil); err != nil {
				fmt.Printf("Template error: %v\n", err)
			}
		})
		mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
			tmpl, err := partials.Clone()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			tmpl, err = tmpl.ParseFiles("web/templates/layout.html", "web/templates/stats.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			if err := tmpl.ExecuteTemplate(w, "layout.html", nil); err != nil {
				fmt.Printf("Template error: %v\n", err)
			}
		})

		mux.HandleFunc("/partials/status", makePartialHandler(database, partials, "status.html", getStatusData))
		mux.HandleFunc("/partials/chart", makePartialHandler(database, partials, "chart.html", getChartData))
		mux.HandleFunc("/partials/burn-rate", makePartialHandler(database, partials, "burn-rate.html", getBurnRateData))
		mux.HandleFunc("/partials/history-table", makePartialHandler(database, partials, "history-table.html", getHistoryData))
		mux.HandleFunc("/partials/daily-stats", makePartialHandler(database, partials, "daily-stats.html", getDailyData))
		mux.HandleFunc("/partials/weekly-stats", makePartialHandler(database, partials, "weekly-stats.html", getWeeklyData))
		mux.HandleFunc("/partials/overall-stats", makePartialHandler(database, partials, "overall-stats.html", getOverallData))

		addr := fmt.Sprintf(":%d", servePort)
		fmt.Printf("Starting server at http://localhost%s\n", addr)
		return http.ListenAndServe(addr, mux)
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}

type partialDataProvider func(database *db.DB) (any, error)

func makePartialHandler(database *db.DB, tmpl *template.Template, name string, provider partialDataProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := provider(database)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		tmpl.ExecuteTemplate(w, name, data)
	}
}

type StatusData struct {
	Limit    int
	Used     int
	Leftover int
	Percent  float64
}

func getStatusData(database *db.DB) (any, error) {
	snapshot, err := database.GetLatestSnapshot()
	if err != nil || snapshot == nil {
		return StatusData{}, err
	}
	return StatusData{
		Limit:    snapshot.SubscriptionLimit,
		Used:     snapshot.RequestsUsed,
		Leftover: snapshot.Leftover,
		Percent:  float64(snapshot.RequestsUsed) / float64(snapshot.SubscriptionLimit) * 100,
	}, nil
}

type ChartPoint struct {
	X float64
	Y float64
}

type ChartLabel struct {
	X          float64
	Y          float64
	Text       string
	FontSize   int
	TextAnchor string
}

type ChartData struct {
	Width       float64
	Height      float64
	Padding     float64
	ChartWidth  float64
	ChartHeight float64
	PointsUsed  []ChartPoint
	PointsLeft  []ChartPoint
	Labels      []ChartLabel
	NoData      bool
}

func getChartData(database *db.DB) (any, error) {
	since := time.Now().AddDate(0, 0, -7)
	snapshots, err := database.GetSnapshots(since)
	if err != nil || len(snapshots) == 0 {
		return ChartData{NoData: true, Width: 800, Height: 400}, nil
	}

	return generateSVGChart(snapshots), nil
}

func generateSVGChart(snapshots []db.UsageSnapshot) ChartData {
	if len(snapshots) < 2 {
		return ChartData{NoData: true, Width: 800, Height: 400}
	}

	width := 800.0
	height := 400.0
	padding := 60.0
	chartWidth := width - 2*padding
	chartHeight := height - 2*padding

	maxVal := float64(snapshots[0].SubscriptionLimit)
	for _, s := range snapshots {
		if float64(s.RequestsUsed) > maxVal {
			maxVal = float64(s.RequestsUsed)
		}
	}

	pointsUsed := make([]ChartPoint, len(snapshots))
	pointsLeft := make([]ChartPoint, len(snapshots))

	for i, s := range snapshots {
		x := padding + (float64(i)/float64(len(snapshots)-1))*chartWidth
		yUsed := padding + chartHeight - (float64(s.RequestsUsed)/maxVal)*chartHeight
		yLeft := padding + chartHeight - (float64(s.Leftover)/maxVal)*chartHeight

		pointsUsed[i] = ChartPoint{X: x, Y: yUsed}
		pointsLeft[i] = ChartPoint{X: x, Y: yLeft}
	}

	step := len(snapshots) / 5
	if step < 1 {
		step = 1
	}

	labels := []ChartLabel{
		{X: padding, Y: padding - 20, Text: "Used", FontSize: 12, TextAnchor: "start"},
		{X: padding + 80, Y: padding - 20, Text: "Leftover", FontSize: 12, TextAnchor: "start"},
	}

	for i, s := range snapshots {
		if i%step == 0 || i == len(snapshots)-1 {
			x := padding + (float64(i)/float64(len(snapshots)-1))*chartWidth
			labels = append(labels, ChartLabel{
				X:          x,
				Y:          height - padding + 20,
				Text:       s.CollectedAt.Format("01/02"),
				FontSize:   10,
				TextAnchor: "middle",
			})
		}
	}

	return ChartData{
		Width:       width,
		Height:      height,
		Padding:     padding,
		ChartWidth:  chartWidth,
		ChartHeight: chartHeight,
		PointsUsed:  pointsUsed,
		PointsLeft:  pointsLeft,
		Labels:      labels,
		NoData:      false,
	}
}

type BurnRateData struct {
	Rate      float64
	HoursLeft float64
	DaysLeft  float64
	HasData   bool
	Leftover  int
}

func getBurnRateData(database *db.DB) (any, error) {
	rate, err := database.GetBurnRate(24)
	if err != nil {
		return BurnRateData{HasData: false}, nil
	}

	latest, err := database.GetLatestSnapshot()
	if err != nil || latest == nil {
		return BurnRateData{HasData: false}, nil
	}

	var hoursLeft float64
	if rate > 0 {
		hoursLeft = float64(latest.Leftover) / rate
	}

	return BurnRateData{
		Rate:      rate,
		HoursLeft: hoursLeft,
		DaysLeft:  hoursLeft / 24,
		HasData:   rate > 0,
		Leftover:  latest.Leftover,
	}, nil
}

type HistoryTableData struct {
	Snapshots []db.UsageSnapshot
}

func getHistoryData(database *db.DB) (any, error) {
	since := time.Now().AddDate(0, 0, -7)
	snapshots, err := database.GetSnapshots(since)
	if err != nil {
		return HistoryTableData{}, err
	}

	for i, j := 0, len(snapshots)-1; i < j; i, j = i+1, j-1 {
		snapshots[i], snapshots[j] = snapshots[j], snapshots[i]
	}

	return HistoryTableData{Snapshots: snapshots}, nil
}

func getDailyData(database *db.DB) (any, error) {
	return database.GetDailyUsage(7)
}

func getWeeklyData(database *db.DB) (any, error) {
	return database.GetWeeklyUsage(4)
}

type OverallData struct {
	TotalSnapshots int
	FirstSnapshot  string
	LatestSnapshot string
	AvgDaily       float64
}

func getOverallData(database *db.DB) (any, error) {
	snapshots, err := database.GetSnapshots(time.Time{})
	if err != nil || len(snapshots) == 0 {
		return OverallData{}, err
	}

	var totalConsumed int
	if len(snapshots) > 1 {
		totalConsumed = snapshots[len(snapshots)-1].RequestsUsed - snapshots[0].RequestsUsed
	}

	days := snapshots[len(snapshots)-1].CollectedAt.Sub(snapshots[0].CollectedAt).Hours() / 24
	var avgDaily float64
	if days > 0 {
		avgDaily = float64(totalConsumed) / days
	}

	return OverallData{
		TotalSnapshots: len(snapshots),
		FirstSnapshot:  snapshots[0].CollectedAt.Format("2006-01-02 15:04"),
		LatestSnapshot: snapshots[len(snapshots)-1].CollectedAt.Format("2006-01-02 15:04"),
		AvgDaily:       avgDaily,
	}, nil
}
