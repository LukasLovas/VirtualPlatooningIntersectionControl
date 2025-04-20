package manager

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"sumo/models"
)

type TrafficManager struct {
	Vehicles          map[string]*models.Vehicle
	Platoons          map[string]*models.Platoon
	Intersections     map[string]*models.Intersection
	VehicleToPlatoon  map[string]string
	TimeStep          int
	DetectionDistance float64
	FollowingGap      float64

	CatchupSpeedFactor float64
	PlatoonGapClose    float64
	PlatoonGapTooClose float64

	MaxRegularSpeed          float64
	MaxPlatoonSpeed          float64
	StablePlatoonSpeed       float64
	IntersectionReservations map[string]*models.IntersectionReservation
	TrafficDensity           map[string]float64
	LastTrafficMeasurement   time.Time

	BenchmarkMode        bool
	BenchmarkName        string
	BenchmarkStartTime   time.Time
	BenchmarkDuration    int
	BenchmarkMetrics     []BenchmarkMetrics
	ThroughputCounter    int
	ThroughputPerStep    int
	TotalCreatedVehicles int
	TotalRemovedVehicles int
	StopBenchmark        bool
	UseCustomAlgorithm   bool
}

func NewTrafficManager() *TrafficManager {
	return &TrafficManager{
		Vehicles:          make(map[string]*models.Vehicle),
		Platoons:          make(map[string]*models.Platoon),
		Intersections:     make(map[string]*models.Intersection),
		VehicleToPlatoon:  make(map[string]string),
		DetectionDistance: 50.0,
		FollowingGap:      10.0,

		CatchupSpeedFactor: 1.3,
		PlatoonGapClose:    15.0,
		PlatoonGapTooClose: 5.0,

		MaxRegularSpeed:          16.7,
		MaxPlatoonSpeed:          19.4,
		StablePlatoonSpeed:       22.2,
		IntersectionReservations: make(map[string]*models.IntersectionReservation),
		TrafficDensity:           make(map[string]float64),
		LastTrafficMeasurement:   time.Now(),

		UseCustomAlgorithm: true,
	}
}

func (tm *TrafficManager) UpdateVehicleData(vehicleData map[string]map[string]interface{}) {
	existingVehicles := make(map[string]bool)

	for id, data := range vehicleData {
		existingVehicles[id] = true

		lane := data["lane"].(string)
		pos := data["pos"].(float64)
		speed := data["speed"].(float64)
		edge := data["edge"].(string)

		if v, exists := tm.Vehicles[id]; exists {
			v.Lane = lane
			v.Pos = pos
			v.Speed = speed
			v.Edge = edge

			v.AtIntersection = tm.isVehicleAtIntersection(v)
		} else {
			tm.Vehicles[id] = &models.Vehicle{
				ID:                id,
				Lane:              lane,
				Pos:               pos,
				Speed:             speed,
				Edge:              edge,
				PlatoonID:         "",
				IsLeader:          false,
				DesiredSpeed:      13.9,
				LeaderID:          "",
				NextEdge:          "",
				TurnDirection:     "",
				AtIntersection:    tm.isVehicleAtIntersection(&models.Vehicle{Edge: edge}),
				LastSpeedChange:   time.Now(),
				StablePlatoonTime: 0,
				ReactionTime:      0.5,
			}
		}
	}

	for id := range tm.Vehicles {
		if !existingVehicles[id] {
			if platoonID, exists := tm.VehicleToPlatoon[id]; exists {
				tm.RemoveVehicleFromPlatoon(id, platoonID)
				delete(tm.VehicleToPlatoon, id)
			}

			for _, intersection := range tm.Intersections {
				for i, vid := range intersection.Vehicles {
					if vid == id {
						intersection.Vehicles = append(intersection.Vehicles[:i],
							intersection.Vehicles[i+1:]...)
						break
					}
				}
			}

			delete(tm.Vehicles, id)
		}
	}

	tm.updateIntersectionStatus()
	tm.measureTrafficDensity()
}

func (tm *TrafficManager) measureTrafficDensity() {
	now := time.Now()
	if now.Sub(tm.LastTrafficMeasurement).Seconds() < 2.0 {
		return
	}

	tm.LastTrafficMeasurement = now
	edgeVehicleCounts := make(map[string]int)

	for _, vehicle := range tm.Vehicles {
		if vehicle.Edge != "" && !vehicle.AtIntersection {
			edgeVehicleCounts[vehicle.Edge]++
		}
	}

	for edge, count := range edgeVehicleCounts {
		length := tm.getEdgeLengths()[edge]
		if length > 0 {
			tm.TrafficDensity[edge] = float64(count) / length * 100
		}
	}
}

func (tm *TrafficManager) updateIntersectionStatus() {
	for _, intersection := range tm.Intersections {
		intersection.Vehicles = []string{}
	}

	for id, vehicle := range tm.Vehicles {
		if vehicle.AtIntersection {
			parts := strings.Split(vehicle.Edge, "_")
			if len(parts) > 0 && len(parts[0]) > 0 && parts[0][0] == ':' {
				intersectionID := parts[0]

				if _, exists := tm.Intersections[intersectionID]; !exists {
					tm.Intersections[intersectionID] = &models.Intersection{
						ID:                  intersectionID,
						InternalID:          intersectionID,
						Edges:               []string{},
						Vehicles:            []string{},
						LastPlatoonPassTime: time.Now().Add(-10 * time.Second),
					}
				}

				tm.Intersections[intersectionID].Vehicles = append(
					tm.Intersections[intersectionID].Vehicles, id)
			}
		}
	}

	tm.cleanExpiredReservations()
}

func (tm *TrafficManager) cleanExpiredReservations() {
	now := time.Now()
	for id, reservation := range tm.IntersectionReservations {
		if now.After(reservation.EndTime) {
			delete(tm.IntersectionReservations, id)
		}
	}
}

func (tm *TrafficManager) Update() {
	tm.TimeStep++

	if tm.UseCustomAlgorithm {
		tm.UpdatePlatoons()
		tm.EstimatePlatoonStability()
		tm.ReservePlatoonIntersectionSlots()
		tm.ManageIntersections()
		tm.SynchronizeSpeeds()
		tm.AdjustSpeedForTrafficDensity()
	} else {
	} //sumo stuff? I guess

	if tm.BenchmarkMode {
		tm.UpdateVehicleThroughput()
		tm.RecordBenchmarkMetrics()
	}

	log.Printf("update: step %d, vehicles: %d, platoons: %d, intersections: %d",
		tm.TimeStep, len(tm.Vehicles), len(tm.Platoons), len(tm.Intersections))
}

func (tm *TrafficManager) EstimatePlatoonStability() {
	for _, platoon := range tm.Platoons {
		if len(platoon.VehicleIDs) < 2 {
			continue
		}

		leader, exists := tm.Vehicles[platoon.LeaderID]
		if !exists {
			continue
		}

		stableCount := 0
		totalVehicles := len(platoon.VehicleIDs)

		for _, vid := range platoon.VehicleIDs {
			if vid == platoon.LeaderID {
				continue
			}

			follower, exists := tm.Vehicles[vid]
			if !exists {
				continue
			}

			if follower.StablePlatoonTime > 5.0 {
				stableCount++
			}
		}

		platoon.StabilityRatio = float64(stableCount) / float64(totalVehicles-1)

		if platoon.StabilityRatio > 0.7 && totalVehicles >= 3 && !leader.AtIntersection {
			leader.DesiredSpeed = tm.StablePlatoonSpeed
		}
	}
}

func (tm *TrafficManager) ReservePlatoonIntersectionSlots() {
	for _, platoon := range tm.Platoons {
		if len(platoon.VehicleIDs) < 3 || platoon.StabilityRatio < 0.6 {
			continue
		}

		leader, exists := tm.Vehicles[platoon.LeaderID]
		if !exists || leader.AtIntersection {
			continue
		}

		nextIntersection := tm.findNextIntersectionForVehicle(leader)
		if nextIntersection == nil {
			continue
		}

		distanceToIntersection := tm.estimateDistanceToIntersection(leader, nextIntersection)
		if distanceToIntersection > 100 || distanceToIntersection < 0 {
			continue
		}

		estimatedArrivalTime := tm.estimateArrivalTime(leader, distanceToIntersection)
		reservationID := fmt.Sprintf("%s_%s", platoon.ID, nextIntersection.ID)

		if _, exists := tm.IntersectionReservations[reservationID]; exists {
			continue
		}

		passingTime := float64(len(platoon.VehicleIDs)) * 1.5
		reservation := &models.IntersectionReservation{
			ID:             reservationID,
			IntersectionID: nextIntersection.ID,
			PlatoonID:      platoon.ID,
			StartTime:      estimatedArrivalTime,
			EndTime:        estimatedArrivalTime.Add(time.Duration(passingTime) * time.Second),
			EdgeFrom:       leader.Edge,
			Direction:      leader.TurnDirection,
		}

		if !tm.hasConflictingReservation(reservation) {
			tm.IntersectionReservations[reservationID] = reservation
			nextIntersection.HasReservation = true
			log.Printf("reserved intersection %s for platoon %s, arrival at %v",
				nextIntersection.ID, platoon.ID, estimatedArrivalTime)
		}
	}
}

func (tm *TrafficManager) hasConflictingReservation(newReservation *models.IntersectionReservation) bool {
	for _, existing := range tm.IntersectionReservations {
		if existing.IntersectionID != newReservation.IntersectionID {
			continue
		}

		if existing.EndTime.Before(newReservation.StartTime) ||
			existing.StartTime.After(newReservation.EndTime) {
			continue
		}

		if tm.areMovementsCompatible(existing.EdgeFrom, existing.Direction,
			newReservation.EdgeFrom, newReservation.Direction) {
			continue
		}

		return true
	}
	return false
}

func (tm *TrafficManager) findNextIntersectionForVehicle(vehicle *models.Vehicle) *models.Intersection {
	routeEdges := tm.getVehicleRouteEdges(vehicle.ID)
	if len(routeEdges) < 2 {
		return nil
	}

	for i, edge := range routeEdges {
		if edge == vehicle.Edge && i < len(routeEdges)-1 {
			for intersectionID, intersection := range tm.Intersections {
				for _, intEdge := range intersection.Edges {
					if intEdge == routeEdges[i+1] || strings.HasPrefix(routeEdges[i+1], ":") {
						return tm.Intersections[intersectionID]
					}
				}
			}
		}
	}

	return nil
}

func (tm *TrafficManager) estimateDistanceToIntersection(vehicle *models.Vehicle, intersection *models.Intersection) float64 {
	edgeLengths := tm.getEdgeLengths()
	if length, exists := edgeLengths[vehicle.Edge]; exists {
		return length - vehicle.Pos
	}
	return -1
}

func (tm *TrafficManager) estimateArrivalTime(vehicle *models.Vehicle, distance float64) time.Time {
	if vehicle.Speed < 1.0 {
		return time.Now().Add(time.Duration(distance / 5.0 * float64(time.Second)))
	}

	return time.Now().Add(time.Duration(distance / vehicle.Speed * float64(time.Second)))
}

func (tm *TrafficManager) AdjustSpeedForTrafficDensity() {
	for id, vehicle := range tm.Vehicles {
		if vehicle.AtIntersection {
			continue
		}

		density, exists := tm.TrafficDensity[vehicle.Edge]
		if !exists {
			continue
		}

		isPlatoonLeader := false
		platoonSize := 0
		if platoonID, hasPlatoon := tm.VehicleToPlatoon[id]; hasPlatoon {
			if platoon, platoonExists := tm.Platoons[platoonID]; platoonExists {
				if platoon.LeaderID == id {
					isPlatoonLeader = true
					platoonSize = len(platoon.VehicleIDs)
				}
			}
		}

		if isPlatoonLeader {
			if density > 70 {
				vehicle.DesiredSpeed = math.Min(vehicle.DesiredSpeed, 8.3)
			} else if density > 50 {
				vehicle.DesiredSpeed = math.Min(vehicle.DesiredSpeed, 11.1)
			} else if density > 30 {
				if platoonSize > 5 {
					vehicle.DesiredSpeed = math.Min(vehicle.DesiredSpeed, tm.MaxPlatoonSpeed)
				} else {
					vehicle.DesiredSpeed = math.Min(vehicle.DesiredSpeed, 16.7)
				}
			} else if platoonSize > 3 && density < 20 {
				vehicle.DesiredSpeed = math.Min(tm.StablePlatoonSpeed, vehicle.DesiredSpeed*1.1)
			}
		} else if !vehicle.IsLeader {
			if density > 70 {
				vehicle.DesiredSpeed = math.Min(vehicle.DesiredSpeed, 7.8)
			}
		}
	}
}

func (tm *TrafficManager) PrepareCommands() map[string]interface{} {
	commands := make(map[string]interface{})

	commands["speeds"] = tm.GetDesiredSpeeds()
	commands["platoons"] = tm.GetPlatoonsForVisualization()
	commands["stats"] = map[string]interface{}{
		"time_step":          tm.TimeStep,
		"vehicle_count":      len(tm.Vehicles),
		"platoon_count":      len(tm.Platoons),
		"intersection_count": len(tm.Intersections),
		"reservations_count": len(tm.IntersectionReservations),
	}

	return commands
}

func (tm *TrafficManager) updateVehicleWaitTimes() {
	for _, vehicle := range tm.Vehicles {
		if vehicle.Speed < 0.5 {
			vehicle.WaitingTime++
		} else {
			vehicle.WaitingTime = 0
		}

		vehicle.TravelTime += 1.0 / 60.0
	}
}

func (tm *TrafficManager) AddVehicle(vehicle *models.Vehicle) {
	tm.Vehicles[vehicle.ID] = vehicle
	tm.TotalCreatedVehicles++
	vehicle.CreationTime = time.Now()
}

func (tm *TrafficManager) RemoveVehicle(vehicleID string) {
	if _, exists := tm.Vehicles[vehicleID]; exists {
		delete(tm.Vehicles, vehicleID)
		tm.TotalRemovedVehicles++
	}
}

func (tm *TrafficManager) RemoveRandomVehicle() string {
	if len(tm.Vehicles) == 0 {
		return ""
	}

	var randomID string
	for id := range tm.Vehicles {
		randomID = id
		break
	}

	tm.RemoveVehicle(randomID)
	return randomID
}

func (tm *TrafficManager) RemoveAllVehicles() int {
	count := len(tm.Vehicles)
	tm.Vehicles = make(map[string]*models.Vehicle)
	tm.TotalRemovedVehicles += count
	return count
}
