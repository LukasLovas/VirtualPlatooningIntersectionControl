package manager

import (
	"fmt"
	"math"
	"strings"
	"time"

	"sumo/models"
)

func (tm *TrafficManager) FindVehicleAhead(vehicle *models.Vehicle) *models.Vehicle {
	var closestVehicle *models.Vehicle
	var minDistance float64 = tm.DetectionDistance + 1

	for _, other := range tm.Vehicles {
		if other.ID == vehicle.ID {
			continue
		}

		if other.Edge != vehicle.Edge || other.Lane != vehicle.Lane {
			continue
		}

		distance := other.Pos - vehicle.Pos
		if distance <= 0 || distance > tm.DetectionDistance {
			continue
		}

		if distance < minDistance {
			minDistance = distance
			closestVehicle = other
		}
	}

	return closestVehicle
}

func (tm *TrafficManager) SynchronizeSpeeds() {
	now := time.Now()

	for id, vehicle := range tm.Vehicles {
		if vehicle.AtIntersection {
			platoonID, inPlatoon := tm.VehicleToPlatoon[id]
			if inPlatoon {
				platoon, platoonExists := tm.Platoons[platoonID]
				if platoonExists && platoon.LeaderID == id {
					reservationID := fmt.Sprintf("%s_%s", platoonID, tm.extractIntersectionID(vehicle.Edge))
					if _, hasReservation := tm.IntersectionReservations[reservationID]; hasReservation {
						vehicle.DesiredSpeed = math.Min(vehicle.Speed+2.0, tm.MaxPlatoonSpeed)
						continue
					}
				}
			}
		}

		if vehicle.LeaderID == "" {
			if platoonID, inPlatoon := tm.VehicleToPlatoon[id]; inPlatoon {
				platoon, platoonExists := tm.Platoons[platoonID]
				if platoonExists && platoon.LeaderID == id {
					if platoon.StabilityRatio > 0.8 && len(platoon.VehicleIDs) > 3 {
						vehicle.DesiredSpeed = math.Min(tm.StablePlatoonSpeed, vehicle.Speed+1.0)
					} else if platoon.StabilityRatio > 0.6 {
						vehicle.DesiredSpeed = math.Min(tm.MaxPlatoonSpeed, vehicle.Speed+0.8)
					} else {
						vehicle.DesiredSpeed = tm.MaxRegularSpeed
					}
				} else {
					vehicle.DesiredSpeed = 13.9
				}
			} else {
				vehicle.DesiredSpeed = 13.9
			}
			continue
		}

		leader, exists := tm.Vehicles[vehicle.LeaderID]
		if !exists {
			vehicle.DesiredSpeed = 13.9
			continue
		}

		currentGap := leader.Pos - vehicle.Pos
		optimalGap := tm.calculateOptimalGap(vehicle, leader)

		desiredSpeed := 0.0
		leaderStopped := leader.Speed < 0.5

		if leaderStopped {
			if currentGap > optimalGap*3.0 {
				desiredSpeed = 16.7
			} else if currentGap > optimalGap*2.0 {
				desiredSpeed = 11.1
			} else if currentGap > optimalGap*1.5 {
				desiredSpeed = 8.3
			} else if currentGap > optimalGap*1.2 {
				desiredSpeed = 5.6
			} else if currentGap > optimalGap*1.05 {
				desiredSpeed = 2.8
			} else if currentGap > optimalGap {
				desiredSpeed = 1.4
			} else {
				desiredSpeed = 0.0
			}
		} else {
			if currentGap > optimalGap*3.0 {
				desiredSpeed = math.Max(22.2, leader.Speed*1.5)
			} else if currentGap > optimalGap*2.0 {
				desiredSpeed = math.Max(19.4, leader.Speed*1.4)
			} else if currentGap > optimalGap*1.5 {
				desiredSpeed = math.Max(16.7, leader.Speed*1.3)
			} else if currentGap > optimalGap*1.1 {
				desiredSpeed = math.Min(leader.Speed*1.1, leader.Speed+2.0)
			} else if currentGap < optimalGap*0.5 {
				desiredSpeed = leader.Speed * 0.5
			} else if currentGap < optimalGap*0.8 {
				desiredSpeed = leader.Speed * 0.85
			} else {
				desiredSpeed = leader.Speed
			}
		}

		vehicle.DesiredSpeed = desiredSpeed

		if vehicle.Speed < 0.5 && currentGap > optimalGap {
			vehicle.DesiredSpeed = math.Max(5.0, desiredSpeed)
		}

		vehicle.DesiredSpeed = math.Max(0.0, vehicle.DesiredSpeed)

		isPlatoonMember := vehicle.PlatoonID != ""
		if isPlatoonMember {
			vehicle.DesiredSpeed = math.Min(tm.MaxPlatoonSpeed, vehicle.DesiredSpeed)
		} else {
			vehicle.DesiredSpeed = math.Min(tm.MaxRegularSpeed, vehicle.DesiredSpeed)
		}

		if math.Abs(vehicle.DesiredSpeed-vehicle.Speed) > 0.5 {
			vehicle.LastSpeedChange = now
		}
	}

	tm.processPlatoonsIndependently()
}

func (tm *TrafficManager) processPlatoonsIndependently() {
	for _, platoon := range tm.Platoons {
		orderedVehicles := tm.getOrderedPlatoonVehicles(platoon)

		if len(orderedVehicles) < 2 {
			continue
		}

		for i, vehicle := range orderedVehicles {
			if i == 0 {
				continue
			}

			frontVehicle := orderedVehicles[i-1]
			currentGap := frontVehicle.Pos - vehicle.Pos

			baseOptimalGap := 7.0
			if platoon.StabilityRatio > 0.6 {
				baseOptimalGap = 5.0
			}

			frontStopped := frontVehicle.Speed < 0.5

			if frontStopped {
				if currentGap > baseOptimalGap*3.0 {
					vehicle.DesiredSpeed = 16.7
				} else if currentGap > baseOptimalGap*2.0 {
					vehicle.DesiredSpeed = 11.1
				} else if currentGap > baseOptimalGap*1.5 {
					vehicle.DesiredSpeed = 8.3
				} else if currentGap > baseOptimalGap*1.2 {
					vehicle.DesiredSpeed = 5.6
				} else if currentGap > baseOptimalGap*1.05 {
					vehicle.DesiredSpeed = 2.8
				} else if currentGap > baseOptimalGap {
					vehicle.DesiredSpeed = 1.4
				} else {
					vehicle.DesiredSpeed = 0.0
				}
			} else {
				if currentGap > baseOptimalGap*2.0 {
					vehicle.DesiredSpeed = math.Max(19.4, frontVehicle.Speed*1.4)
				} else if currentGap > baseOptimalGap*1.5 {
					vehicle.DesiredSpeed = math.Max(16.7, frontVehicle.Speed*1.3)
				} else if currentGap > baseOptimalGap*1.2 {
					vehicle.DesiredSpeed = frontVehicle.Speed * 1.2
				} else if currentGap < baseOptimalGap*0.6 {
					vehicle.DesiredSpeed = frontVehicle.Speed * 0.6
				} else if currentGap < baseOptimalGap*0.8 {
					vehicle.DesiredSpeed = frontVehicle.Speed * 0.8
				} else {
					vehicle.DesiredSpeed = frontVehicle.Speed
				}
			}

			if vehicle.Speed < 0.5 && currentGap > baseOptimalGap {
				vehicle.DesiredSpeed = math.Max(5.0, vehicle.DesiredSpeed)
			}
		}
	}
}

func (tm *TrafficManager) getOrderedPlatoonVehicles(platoon *models.Platoon) []*models.Vehicle {
	vehicles := make([]*models.Vehicle, 0, len(platoon.VehicleIDs))
	for _, id := range platoon.VehicleIDs {
		if v, exists := tm.Vehicles[id]; exists {
			vehicles = append(vehicles, v)
		}
	}

	if len(vehicles) <= 1 {
		return vehicles
	}

	for i := 0; i < len(vehicles)-1; i++ {
		for j := 0; j < len(vehicles)-i-1; j++ {
			if vehicles[j].Pos < vehicles[j+1].Pos {
				vehicles[j], vehicles[j+1] = vehicles[j+1], vehicles[j]
			}
		}
	}

	return vehicles
}

func (tm *TrafficManager) calculateOptimalGap(follower, leader *models.Vehicle) float64 {
	baseGap := tm.FollowingGap

	speedFactor := follower.Speed / 10.0
	if speedFactor > 1.0 {
		baseGap *= speedFactor
	}

	isPlatoonMember := follower.PlatoonID != ""
	if isPlatoonMember && follower.StablePlatoonTime > 5.0 {
		baseGap *= 0.7
	}

	platoonID, inPlatoon := tm.VehicleToPlatoon[follower.ID]
	if inPlatoon {
		platoon, platoonExists := tm.Platoons[platoonID]
		if platoonExists && platoon.StabilityRatio > 0.7 {
			baseGap *= 0.8
		}
	}

	timeGap := follower.ReactionTime * follower.Speed

	return math.Max(5.0, math.Min(baseGap, timeGap))
}

func (tm *TrafficManager) extractIntersectionID(edge string) string {
	if len(edge) > 0 && edge[0] == ':' {
		parts := strings.Split(edge, "_")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

func (tm *TrafficManager) GetDesiredSpeeds() map[string]float64 {
	speeds := make(map[string]float64)
	for id, vehicle := range tm.Vehicles {
		speeds[id] = vehicle.DesiredSpeed
	}
	return speeds
}
