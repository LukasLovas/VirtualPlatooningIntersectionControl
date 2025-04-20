package manager

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type BenchmarkMetrics struct {
	TimeStep              int     `json:"timeStep"`
	TotalVehicles         int     `json:"totalVehicles"`
	AverageSpeed          float64 `json:"averageSpeed"`
	AverageWaitTime       float64 `json:"averageWaitTime"`
	MaxWaitTime           int     `json:"maxWaitTime"`
	MinWaitTime           int     `json:"minWaitTime"`
	IntersectionQueueSize int     `json:"intersectionQueueSize"`
	ThroughputCount       int     `json:"throughputCount"`
	TotalThroughput       int     `json:"totalThroughput"`
	PlatoonCount          int     `json:"platoonCount"`
	AveragePlatoonSize    float64 `json:"averagePlatoonSize"`
	MaxPlatoonSize        int     `json:"maxPlatoonSize"`
	TotalCreatedVehicles  int     `json:"totalCreatedVehicles"`
	TotalRemovedVehicles  int     `json:"totalRemovedVehicles"`
	AverageTravelTime     float64 `json:"averageTravelTime"`
	MaxTravelTime         float64 `json:"maxTravelTime"`
	TrafficDensity        float64 `json:"trafficDensity"`
	SimulationTimeElapsed float64 `json:"simulationTimeElapsed"`
	CPUUsage              float64 `json:"cpuUsage"`
}

type SimulationSummary struct {
	AlgorithmType            string  `json:"algorithmType"`
	TotalSteps               int     `json:"totalSteps"`
	AverageVehicles          float64 `json:"averageVehicles"`
	TotalUniqueVehicles      int     `json:"totalUniqueVehicles"`
	FinalThroughput          int     `json:"finalThroughput"`
	AverageSpeed             float64 `json:"averageSpeed"`
	AverageWaitTime          float64 `json:"averageWaitTime"`
	MaxWaitTime              int     `json:"maxWaitTime"`
	AverageTravelTime        float64 `json:"averageTravelTime"`
	MaxTravelTime            float64 `json:"maxTravelTime"`
	AverageIntersectionQueue float64 `json:"averageIntersectionQueue"`
	AveragePlatoonSize       float64 `json:"averagePlatoonSize"`
	MaxPlatoonSize           int     `json:"maxPlatoonSize"`
	AverageTrafficDensity    float64 `json:"averageTrafficDensity"`
	SimulationRuntime        float64 `json:"simulationRuntime"`
	Timestamp                string  `json:"timestamp"`
}

func (tm *TrafficManager) StartBenchmark(duration int, name string) {
	tm.BenchmarkMode = true
	tm.BenchmarkName = name
	tm.BenchmarkStartTime = time.Now()
	tm.BenchmarkDuration = duration
	tm.BenchmarkMetrics = make([]BenchmarkMetrics, 0)
	tm.ThroughputCounter = 0
	tm.TotalCreatedVehicles = 0
	tm.TotalRemovedVehicles = 0

	os.MkdirAll("statistics", 0755)

	log.Printf("starting %s benchmark for %d steps", name, duration)
}

func (tm *TrafficManager) RecordBenchmarkMetrics() {
	if !tm.BenchmarkMode {
		return
	}

	maxWaitTime, minWaitTime := tm.calculateWaitTimeStats()
	maxPlatoonSize := tm.calculateMaxPlatoonSize()
	avgTravelTime, maxTravelTime := tm.calculateTravelTimeStats()

	metrics := BenchmarkMetrics{
		TimeStep:              tm.TimeStep,
		TotalVehicles:         len(tm.Vehicles),
		AverageSpeed:          tm.CalculateAverageSpeed(),
		AverageWaitTime:       tm.calculateAverageWaitTime(),
		MaxWaitTime:           maxWaitTime,
		MinWaitTime:           minWaitTime,
		IntersectionQueueSize: tm.calculateIntersectionQueueSize(),
		ThroughputCount:       tm.ThroughputPerStep,
		TotalThroughput:       tm.ThroughputCounter,
		PlatoonCount:          len(tm.Platoons),
		AveragePlatoonSize:    tm.calculateAveragePlatoonSize(),
		MaxPlatoonSize:        maxPlatoonSize,
		TotalCreatedVehicles:  tm.TotalCreatedVehicles,
		TotalRemovedVehicles:  tm.TotalRemovedVehicles,
		AverageTravelTime:     avgTravelTime,
		MaxTravelTime:         maxTravelTime,
		TrafficDensity:        tm.calculateTrafficDensity(),
		SimulationTimeElapsed: time.Since(tm.BenchmarkStartTime).Seconds(),
		CPUUsage:              calculateCPUUsage(),
	}

	tm.BenchmarkMetrics = append(tm.BenchmarkMetrics, metrics)

	tm.ThroughputPerStep = 0

	if tm.TimeStep >= tm.BenchmarkDuration || tm.StopBenchmark {
		tm.SaveBenchmarkResults()
		tm.BenchmarkMode = false
		log.Printf("benchmark completed: %s", tm.BenchmarkName)
	}
}

func (tm *TrafficManager) CalculateAverageSpeed() float64 {
	if len(tm.Vehicles) == 0 {
		return 0
	}

	totalSpeed := 0.0
	for _, v := range tm.Vehicles {
		totalSpeed += v.Speed
	}

	return totalSpeed / float64(len(tm.Vehicles))
}

func (tm *TrafficManager) calculateWaitTimeStats() (int, int) {
	maxWait := 0
	minWait := -1

	for _, v := range tm.Vehicles {
		if v.WaitingTime > 0 {
			if v.WaitingTime > maxWait {
				maxWait = v.WaitingTime
			}
			if minWait == -1 || v.WaitingTime < minWait {
				minWait = v.WaitingTime
			}
		}
	}

	if minWait == -1 {
		minWait = 0
	}

	return maxWait, minWait
}

func (tm *TrafficManager) calculateAverageWaitTime() float64 {
	totalWaitTime := 0
	waitingVehicles := 0

	for _, v := range tm.Vehicles {
		if v.WaitingTime > 0 {
			totalWaitTime += v.WaitingTime
			waitingVehicles++
		}
	}

	if waitingVehicles == 0 {
		return 0
	}

	return float64(totalWaitTime) / float64(waitingVehicles)
}

func (tm *TrafficManager) calculateIntersectionQueueSize() int {
	count := 0
	for _, intersection := range tm.Intersections {
		count += len(intersection.Vehicles)
	}
	return count
}

func (tm *TrafficManager) calculateAveragePlatoonSize() float64 {
	if len(tm.Platoons) == 0 {
		return 0
	}

	totalSize := 0
	for _, platoon := range tm.Platoons {
		totalSize += len(platoon.VehicleIDs)
	}

	return float64(totalSize) / float64(len(tm.Platoons))
}

func (tm *TrafficManager) calculateMaxPlatoonSize() int {
	maxSize := 0

	for _, platoon := range tm.Platoons {
		if len(platoon.VehicleIDs) > maxSize {
			maxSize = len(platoon.VehicleIDs)
		}
	}

	return maxSize
}

func (tm *TrafficManager) calculateTravelTimeStats() (float64, float64) {
	totalTime := 0.0
	maxTime := 0.0
	count := 0

	for _, v := range tm.Vehicles {
		if v.TravelTime > 0 {
			totalTime += v.TravelTime
			if v.TravelTime > maxTime {
				maxTime = v.TravelTime
			}
			count++
		}
	}

	if count == 0 {
		return 0, 0
	}

	return totalTime / float64(count), maxTime
}

func (tm *TrafficManager) calculateTrafficDensity() float64 {
	totalLength := 0.0
	for _, length := range tm.getEdgeLengths() {
		totalLength += length
	}

	if totalLength == 0 {
		return 0
	}

	return float64(len(tm.Vehicles)) / totalLength
}

func calculateCPUUsage() float64 {
	//todo?idklol maybe
	return 0.0
}

func (tm *TrafficManager) UpdateVehicleThroughput() {
	for _, vehicle := range tm.Vehicles {
		if tm.isLeavingEdge(vehicle.Edge) && !vehicle.CountedInThroughput {
			tm.ThroughputCounter++
			tm.ThroughputPerStep++
			vehicle.CountedInThroughput = true
		}
	}
}

func (tm *TrafficManager) isLeavingEdge(edge string) bool {
	switch edge {
	case "left_leaving", "right_leaving", "up_leaving", "down_leaving":
		return true
	default:
		return false
	}
}

func (tm *TrafficManager) SaveBenchmarkResults() {
	if len(tm.BenchmarkMetrics) == 0 {
		log.Println("no benchmark metrics to save")
		return
	}

	summary := tm.createSimulationSummary()

	csvFilename := fmt.Sprintf("statistics/benchmark_%s_%s.csv",
		tm.BenchmarkName, time.Now().Format("20060102_150405"))

	file, err := os.Create(csvFilename)
	if err != nil {
		log.Printf("failed to create benchmark CSV file: %v", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"TimeStep",
		"TotalVehicles",
		"AverageSpeed",
		"AverageWaitTime",
		"MaxWaitTime",
		"MinWaitTime",
		"IntersectionQueueSize",
		"ThroughputCount",
		"TotalThroughput",
		"PlatoonCount",
		"AveragePlatoonSize",
		"MaxPlatoonSize",
		"TotalCreatedVehicles",
		"TotalRemovedVehicles",
		"AverageTravelTime",
		"MaxTravelTime",
		"TrafficDensity",
		"SimulationTimeElapsed",
		"CPUUsage",
	}

	if err := writer.Write(header); err != nil {
		log.Printf("failed to write CSV header: %v", err)
		return
	}

	for _, m := range tm.BenchmarkMetrics {
		record := []string{
			fmt.Sprintf("%d", m.TimeStep),
			fmt.Sprintf("%d", m.TotalVehicles),
			fmt.Sprintf("%.2f", m.AverageSpeed),
			fmt.Sprintf("%.2f", m.AverageWaitTime),
			fmt.Sprintf("%d", m.MaxWaitTime),
			fmt.Sprintf("%d", m.MinWaitTime),
			fmt.Sprintf("%d", m.IntersectionQueueSize),
			fmt.Sprintf("%d", m.ThroughputCount),
			fmt.Sprintf("%d", m.TotalThroughput),
			fmt.Sprintf("%d", m.PlatoonCount),
			fmt.Sprintf("%.2f", m.AveragePlatoonSize),
			fmt.Sprintf("%d", m.MaxPlatoonSize),
			fmt.Sprintf("%d", m.TotalCreatedVehicles),
			fmt.Sprintf("%d", m.TotalRemovedVehicles),
			fmt.Sprintf("%.2f", m.AverageTravelTime),
			fmt.Sprintf("%.2f", m.MaxTravelTime),
			fmt.Sprintf("%.5f", m.TrafficDensity),
			fmt.Sprintf("%.2f", m.SimulationTimeElapsed),
			fmt.Sprintf("%.2f", m.CPUUsage),
		}

		if err := writer.Write(record); err != nil {
			log.Printf("failed to write CSV record: %v", err)
			continue
		}
	}

	jsonFilename := fmt.Sprintf("statistics/summary_%s_%s.json",
		tm.BenchmarkName, time.Now().Format("20060102_150405"))

	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Printf("failed to marshal JSON summary: %v", err)
		return
	}

	if err := os.WriteFile(jsonFilename, jsonData, 0644); err != nil {
		log.Printf("ailed to write JSON summary file: %v", err)
		return
	}

	log.Printf("benchmark results saved to %s and %s", csvFilename, jsonFilename)
}

func (tm *TrafficManager) createSimulationSummary() SimulationSummary {
	if len(tm.BenchmarkMetrics) == 0 {
		return SimulationSummary{}
	}

	totalVehicles := 0
	totalSpeed := 0.0
	totalWaitTime := 0.0
	totalQueue := 0
	totalPlatoonSize := 0.0
	totalTrafficDensity := 0.0

	maxWaitTime := 0
	maxTravelTime := 0.0
	maxPlatoonSize := 0

	for _, m := range tm.BenchmarkMetrics {
		totalVehicles += m.TotalVehicles
		totalSpeed += m.AverageSpeed
		totalWaitTime += m.AverageWaitTime
		totalQueue += m.IntersectionQueueSize
		totalPlatoonSize += m.AveragePlatoonSize
		totalTrafficDensity += m.TrafficDensity

		if m.MaxWaitTime > maxWaitTime {
			maxWaitTime = m.MaxWaitTime
		}
		if m.MaxTravelTime > maxTravelTime {
			maxTravelTime = m.MaxTravelTime
		}
		if m.MaxPlatoonSize > maxPlatoonSize {
			maxPlatoonSize = m.MaxPlatoonSize
		}
	}

	stepCount := len(tm.BenchmarkMetrics)
	finalMetrics := tm.BenchmarkMetrics[len(tm.BenchmarkMetrics)-1]

	return SimulationSummary{
		AlgorithmType:            tm.BenchmarkName,
		TotalSteps:               tm.TimeStep,
		AverageVehicles:          float64(totalVehicles) / float64(stepCount),
		TotalUniqueVehicles:      tm.TotalCreatedVehicles,
		FinalThroughput:          finalMetrics.TotalThroughput,
		AverageSpeed:             totalSpeed / float64(stepCount),
		AverageWaitTime:          totalWaitTime / float64(stepCount),
		MaxWaitTime:              maxWaitTime,
		AverageTravelTime:        finalMetrics.AverageTravelTime,
		MaxTravelTime:            maxTravelTime,
		AverageIntersectionQueue: float64(totalQueue) / float64(stepCount),
		AveragePlatoonSize:       totalPlatoonSize / float64(stepCount),
		MaxPlatoonSize:           maxPlatoonSize,
		AverageTrafficDensity:    totalTrafficDensity / float64(stepCount),
		SimulationRuntime:        finalMetrics.SimulationTimeElapsed,
		Timestamp:                time.Now().Format("2006-01-02T15:04:05"),
	}
}
