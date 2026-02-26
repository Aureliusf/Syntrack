package cmd

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aure/syntrack/internal/config"
	"github.com/aure/syntrack/internal/db"
	"github.com/spf13/cobra"
)

var servePort int
var requireAuth bool
var authTokens []string
var bindAll bool
var useTailscale bool
var tailscaleIP string
var serveSilent bool

func detectTailscaleIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if !strings.Contains(strings.ToLower(iface.Name), "tailscale") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				ip := ipnet.IP
				if ip.To4() != nil && !ip.IsLoopback() {
					return ip.String()
				}
			}
		}
	}

	return ""
}

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

		// Check X-Auth-Token header or token query parameter
		token := r.Header.Get("X-Auth-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
			if token == "" {
				http.Error(w, "Unauthorized: X-Auth-Token header or token query parameter required", http.StatusUnauthorized)
				return
			}
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
		// Handle silent mode: start server in background
		if serveSilent {
			return startServerInBackground()
		}

		// Load config to get auth tokens
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Load auth tokens if auth is required
		if requireAuth || bindAll || useTailscale {
			authTokens = cfg.AuthTokens
			if len(authTokens) == 0 {
				return fmt.Errorf("external access requires authentication; set SYNTRACK_AUTH_TOKENS or use 'syntrack token generate --save'")
			}
			requireAuth = true // Force auth when binding externally
		}

		var bindHost string
		if bindAll {
			bindHost = "0.0.0.0"
		} else if useTailscale {
			bindHost = tailscaleIP
			if bindHost == "" {
				detected := detectTailscaleIP()
				if detected == "" {
					fmt.Println("Warning: Tailscale interface not detected. Use --tailscale-ip to specify manually.")
					bindHost = "127.0.0.1"
				} else {
					fmt.Printf("Detected Tailscale IP: %s\n", detected)
					bindHost = detected
				}
			}
		} else {
			bindHost = "127.0.0.1"
		}

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

		// Apply token auth middleware
		handler := tokenAuth(mux)

		addr := fmt.Sprintf("%s:%d", bindHost, servePort)
		fmt.Printf("Starting server at http://%s\n", addr)
		if requireAuth {
			fmt.Printf("Authentication enabled with %d token(s)\n", len(authTokens))
			fmt.Println("External requests require X-Auth-Token header or token query parameter")
		}
		return http.ListenAndServe(addr, handler)
	},
}

func startServerInBackground() error {
	fmt.Println("Starting syntrack server in the background...")

	// Get the current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	// Build arguments for the background process (without --silent flag)
	args := []string{"serve"}
	if servePort != 8080 {
		args = append(args, "-p", fmt.Sprintf("%d", servePort))
	}
	if requireAuth {
		args = append(args, "--auth")
	}
	if bindAll {
		args = append(args, "--bind-all")
	}
	if useTailscale {
		args = append(args, "--tailscale")
	}
	if tailscaleIP != "" {
		args = append(args, "--tailscale-ip", tailscaleIP)
	}

	// Start the server process detached from parent
	cmd := exec.Command(exePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = nil

	// Detach from parent process
	if cmd.SysProcAttr != nil {
		cmd.SysProcAttr.Setpgid = true
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting server process: %w", err)
	}

	// Get the bind host for the health check
	var bindHost string
	if bindAll {
		bindHost = "0.0.0.0"
	} else if useTailscale {
		bindHost = tailscaleIP
		if bindHost == "" {
			bindHost = detectTailscaleIP()
			if bindHost == "" {
				bindHost = "127.0.0.1"
			}
		}
	} else {
		bindHost = "127.0.0.1"
	}

	addr := fmt.Sprintf("%s:%d", bindHost, servePort)
	url := fmt.Sprintf("http://%s", addr)

	fmt.Printf("Server starting on %s\n", url)
	fmt.Println("Waiting for server to be ready...")

	// Wait and check if server is ready
	client := &http.Client{Timeout: 2 * time.Second}
	maxAttempts := 15
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(500 * time.Millisecond)
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			fmt.Printf("✓ Server is online and reachable at %s\n", url)
			fmt.Println("The server is now running in the background.")
			fmt.Printf("Process ID: %d\n", cmd.Process.Pid)
			return nil
		}
	}

	// If we get here, server didn't start in time
	return fmt.Errorf("server failed to start within %d seconds", maxAttempts/2)
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "Port to listen on")
	serveCmd.Flags().BoolVar(&requireAuth, "auth", false, "Require token authentication (reads from SYNTRACK_AUTH_TOKENS env or ~/.syntrack/tokens)")
	serveCmd.Flags().BoolVar(&bindAll, "bind-all", false, "Bind to all interfaces (0.0.0.0) - requires auth token")
	serveCmd.Flags().BoolVar(&useTailscale, "tailscale", false, "Bind to Tailscale interface")
	serveCmd.Flags().StringVar(&tailscaleIP, "tailscale-ip", "", "Tailscale IP address (auto-detected if not specified)")
	serveCmd.Flags().BoolVar(&serveSilent, "silent", false, "Start server in background and exit")
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

type ChartData struct {
	SVGContent template.HTML
}

func getChartData(database *db.DB) (any, error) {
	since := time.Now().AddDate(0, 0, -7)
	snapshots, err := database.GetSnapshots(since)
	if err != nil || len(snapshots) == 0 {
		return ChartData{SVGContent: template.HTML("<text x='400' y='200' text-anchor='middle' fill='#71767b'>No data available</text>")}, nil
	}

	svg := generateSVGChart(snapshots)
	return ChartData{SVGContent: template.HTML(svg)}, nil
}

func generateSVGChart(snapshots []db.UsageSnapshot) string {
	if len(snapshots) < 2 {
		return "<text x='400' y='200' text-anchor='middle' fill='#71767b'>Need more data points</text>"
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

	pointsUsed := make([]string, len(snapshots))
	pointsLeft := make([]string, len(snapshots))

	for i, s := range snapshots {
		x := padding + (float64(i)/float64(len(snapshots)-1))*chartWidth
		yUsed := padding + chartHeight - (float64(s.RequestsUsed)/maxVal)*chartHeight
		yLeft := padding + chartHeight - (float64(s.Leftover)/maxVal)*chartHeight

		pointsUsed[i] = fmt.Sprintf("%.1f,%.1f", x, yUsed)
		pointsLeft[i] = fmt.Sprintf("%.1f,%.1f", x, yLeft)
	}

	var svg strings.Builder
	svg.WriteString(fmt.Sprintf(`<svg viewBox="0 0 %.0f %.0f" xmlns="http://www.w3.org/2000/svg">`, width, height))

	svg.WriteString(`<rect width="100%" height="100%" fill="#0f1419"/>`)

	svg.WriteString(fmt.Sprintf(`<line x1="%.0f" y1="%.0f" x2="%.0f" y2="%.0f" stroke="#2f3336" stroke-width="1"/>`, padding, padding, padding, height-padding))
	svg.WriteString(fmt.Sprintf(`<line x1="%.0f" y1="%.0f" x2="%.0f" y2="%.0f" stroke="#2f3336" stroke-width="1"/>`, padding, height-padding, width-padding, height-padding))

	svg.WriteString(fmt.Sprintf(`<text x="%.0f" y="%.0f" fill="#71767b" font-size="12">Used</text>`, padding, padding-20))
	svg.WriteString(fmt.Sprintf(`<text x="%.0f" y="%.0f" fill="#71767b" font-size="12">Leftover</text>`, padding+80, padding-20))

	svg.WriteString(fmt.Sprintf(`<polyline points="%s" fill="none" stroke="#f4212e" stroke-width="2"/>`, strings.Join(pointsUsed, " ")))
	svg.WriteString(fmt.Sprintf(`<polyline points="%s" fill="none" stroke="#00ba7c" stroke-width="2"/>`, strings.Join(pointsLeft, " ")))

	step := len(snapshots) / 5
	if step < 1 {
		step = 1
	}
	for i, s := range snapshots {
		if i%step == 0 || i == len(snapshots)-1 {
			x := padding + (float64(i)/float64(len(snapshots)-1))*chartWidth
			label := s.CollectedAt.Format("01/02")
			svg.WriteString(fmt.Sprintf(`<text x="%.0f" y="%.0f" fill="#71767b" font-size="10" text-anchor="middle">%s</text>`, x, height-padding+20, label))
		}
	}

	svg.WriteString(`</svg>`)
	return svg.String()
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
