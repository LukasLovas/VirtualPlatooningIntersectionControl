package web

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sumo/manager"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebServer struct {
	TrafficManager *manager.TrafficManager
	SumoConn       net.Conn
	clients        map[*websocket.Conn]bool
	clientsMutex   sync.Mutex
	serverMutex    sync.Mutex
}

func NewWebServer(tm *manager.TrafficManager) *WebServer {
	return &WebServer{
		TrafficManager: tm,
		clients:        make(map[*websocket.Conn]bool),
	}
}

func (s *WebServer) SetSumoConnection(conn net.Conn) {
	s.SumoConn = conn
}

func (s *WebServer) Start() {
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", s.handleHome)
	http.HandleFunc("/ws", s.handleWs)
	http.HandleFunc("/api/metrics", s.handleMetrics)
	http.HandleFunc("/api/stats", s.handleStats)
	http.HandleFunc("/api/control", s.handleControl)
	http.HandleFunc("/api/csv-data", s.handleCsvData)

	go s.broadcastMetrics()

	log.Printf("web server starting on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("failed to start web server: %v", err)
	}
}

func (s *WebServer) handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		log.Printf("err parsing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		log.Printf("err executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (s *WebServer) handleWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("err upgrading connection: %v", err)
		return
	}

	s.clientsMutex.Lock()
	s.clients[conn] = true
	s.clientsMutex.Unlock()

	log.Printf("new webSocket client connected. Total clients: %d", len(s.clients))

	go s.handleClientMessages(conn)
}

func (s *WebServer) handleClientMessages(conn *websocket.Conn) {
	defer func() {
		conn.Close()
		s.clientsMutex.Lock()
		delete(s.clients, conn)
		log.Printf("webSocket client disconnected. Remaining clients: %d", len(s.clients))
		s.clientsMutex.Unlock()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("webSocket error: %v", err)
			}
			break
		}
	}
}

func (s *WebServer) broadcastMetrics() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if len(s.clients) == 0 {
			continue
		}

		metrics := s.collectMetrics()
		metricsJson, err := json.Marshal(metrics)
		if err != nil {
			log.Printf("err marshaling metrics: %v", err)
			continue
		}

		s.clientsMutex.Lock()
		for conn := range s.clients {
			err := conn.WriteMessage(websocket.TextMessage, metricsJson)
			if err != nil {
				log.Printf("webSocket write error: %v", err)
				conn.Close()
				delete(s.clients, conn)
			}
		}
		s.clientsMutex.Unlock()
	}
}

func (s *WebServer) collectMetrics() map[string]interface{} {
	s.serverMutex.Lock()
	defer s.serverMutex.Unlock()

	tm := s.TrafficManager

	metrics := map[string]interface{}{
		"time_step":          tm.TimeStep,
		"vehicle_count":      len(tm.Vehicles),
		"platoon_count":      len(tm.Platoons),
		"intersection_count": len(tm.Intersections),
		"average_speed":      tm.CalculateAverageSpeed(),
		"total_throughput":   tm.ThroughputCounter,
		"using_custom_algo":  tm.UseCustomAlgorithm,
	}

	if tm.BenchmarkMode {
		metrics["benchmark_mode"] = true
		metrics["benchmark_name"] = tm.BenchmarkName

		if len(tm.BenchmarkMetrics) > 0 {
			lastMetric := tm.BenchmarkMetrics[len(tm.BenchmarkMetrics)-1]
			metrics["average_wait_time"] = lastMetric.AverageWaitTime
			metrics["max_wait_time"] = lastMetric.MaxWaitTime
			metrics["intersection_queue_size"] = lastMetric.IntersectionQueueSize
			metrics["throughput_per_step"] = lastMetric.ThroughputCount
			metrics["platoon_count"] = lastMetric.PlatoonCount
			metrics["average_platoon_size"] = lastMetric.AveragePlatoonSize
			metrics["max_platoon_size"] = lastMetric.MaxPlatoonSize
			metrics["traffic_density"] = lastMetric.TrafficDensity
		}
	}

	return metrics
}

func (s *WebServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := s.collectMetrics()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(metrics)
}

func (s *WebServer) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"files": s.getStatisticsFiles(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *WebServer) getStatisticsFiles() []map[string]string {
	files := []map[string]string{}

	filepath.Walk("statistics", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(path, ".csv") {
			webPath := strings.ReplaceAll(path, "\\", "/")

			algoType := "unknown"
			if strings.Contains(path, "custom") {
				algoType = "custom"
			} else if strings.Contains(path, "sumo") {
				algoType = "sumo"
			}

			files = append(files, map[string]string{
				"path": webPath,
				"name": info.Name(),
				"size": fmt.Sprintf("%.2f KB", float64(info.Size())/1024),
				"time": info.ModTime().Format("2006-01-02 15:04:05"),
				"algo": algoType,
			})
		}

		return nil
	})

	return files
}

func (s *WebServer) handleControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.serverMutex.Lock()
	defer s.serverMutex.Unlock()

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		if err := r.ParseForm(); err != nil {
			log.Printf("failed to parse form data: %v", err)
		}
	}

	action := r.FormValue("action")
	if action == "" {
		log.Printf("control request missing action parameter: %v", r.PostForm)
		http.Error(w, "Action parameter required", http.StatusBadRequest)
		return
	}

	log.Printf("control action received: %s", action)
	result := map[string]interface{}{"success": true}

	switch action {
	case "change_algo":
		algo := r.FormValue("algorithm")
		if algo == "" {
			algo = "custom"
		}

		currentAlgoType := "sumo"
		if s.TrafficManager.UseCustomAlgorithm {
			currentAlgoType = "custom"
		}

		if s.TrafficManager.BenchmarkMode && len(s.TrafficManager.BenchmarkMetrics) > 0 {
			s.TrafficManager.SaveBenchmarkResults()
		}

		s.TrafficManager.UseCustomAlgorithm = (algo == "custom")

		duration := 1000
		s.TrafficManager.StartBenchmark(duration, algo)

		result["message"] = fmt.Sprintf("changed from %s to %s algorithm", currentAlgoType, algo)
		log.Printf("changed from %s to %s algorithm", currentAlgoType, algo)

	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *WebServer) handleCsvData(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		http.Error(w, "File parameter required", http.StatusBadRequest)
		return
	}

	filePath = strings.ReplaceAll(filePath, "/", string(os.PathSeparator))

	statsDir, err := filepath.Abs("statistics")
	if err != nil {
		log.Printf("err getting absolute path for statistics directory: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Printf("invalid file path: %s", filePath)
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(absPath, statsDir) {
		log.Printf("security violation: path %s not under statistics directory", absPath)
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("failed to open CSV file %s: %v", filePath, err)
		http.Error(w, fmt.Sprintf("File not found: %s", err.Error()), http.StatusNotFound)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		log.Printf("err reading CSV headers from %s: %v", filePath, err)
		http.Error(w, fmt.Sprintf("Error reading CSV headers: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	var rows []map[string]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("err reading CSV row: %v", err)
			continue
		}

		row := make(map[string]string)
		for i, value := range record {
			if i < len(headers) {
				row[headers[i]] = value
			}
		}
		rows = append(rows, row)
	}

	algoType := "unknown"
	if strings.Contains(filePath, "custom") {
		algoType = "custom"
	} else if strings.Contains(filePath, "sumo") {
		algoType = "sumo"
	}

	data := map[string]interface{}{
		"headers": headers,
		"rows":    rows,
		"path":    filePath,
		"algo":    algoType,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)

	log.Printf("served CSV data for %s: %d rows", filePath, len(rows))
}
