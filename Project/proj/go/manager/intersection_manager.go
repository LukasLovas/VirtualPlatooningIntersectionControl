package manager

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"sumo/models"
)

func (tm *TrafficManager) isVehicleAtIntersection(vehicle *models.Vehicle) bool {
	if len(vehicle.Edge) > 0 && vehicle.Edge[0] == ':' {
		return true
	}

	edgeLengths := tm.getEdgeLengths()
	if length, exists := edgeLengths[vehicle.Edge]; exists {
		distanceToEnd := length - vehicle.Pos
		if distanceToEnd < 15.0 {
			return true
		}
	}

	if tm.isEdgeConnectedToJunction(vehicle.Edge) {
		if edgeLength, exists := edgeLengths[vehicle.Edge]; exists {
			remainingDistance := edgeLength - vehicle.Pos

			if remainingDistance < 20.0 {
				return true
			}
		}
	}

	if vehicle.Speed < 5.0 && tm.isEdgeConnectedToJunction(vehicle.Edge) {
		return true
	}

	return false
}

func (tm *TrafficManager) getEdgeLengths() map[string]float64 {
	lengths := map[string]float64{
		"down_incoming":  126.23,
		"down_leaving":   126.10,
		"left_incoming":  128.80,
		"left_leaving":   124.29,
		"right_incoming": 124.29,
		"right_leaving":  128.80,
		"up_incoming":    126.10,
		"up_leaving":     126.23,
	}

	return lengths
}

func (tm *TrafficManager) isEdgeConnectedToJunction(edgeID string) bool {
	connectsToJunction := map[string]bool{
		"down_incoming":  true,
		"down_leaving":   true,
		"left_incoming":  true,
		"left_leaving":   true,
		"right_incoming": true,
		"right_leaving":  true,
		"up_incoming":    true,
		"up_leaving":     true,
	}

	return connectsToJunction[edgeID]
}

func (tm *TrafficManager) determineTurnDirection(vehicle *models.Vehicle, nextEdge string) string {
	currentEdge := vehicle.Edge

	if len(currentEdge) > 0 && currentEdge[0] == ':' {
		if strings.Contains(vehicle.Lane, "_left") {
			return models.TurnLeft
		} else if strings.Contains(vehicle.Lane, "_right") {
			return models.TurnRight
		} else {
			leftTurnPattern := regexp.MustCompile(`(?i)_l[0-9]?$|_sl[0-9]?$|left`)
			rightTurnPattern := regexp.MustCompile(`(?i)_r[0-9]?$|_sr[0-9]?$|right`)

			if leftTurnPattern.MatchString(vehicle.Lane) {
				return models.TurnLeft
			}
			if rightTurnPattern.MatchString(vehicle.Lane) {
				return models.TurnRight
			}

			return models.TurnStraight
		}
	}

	if nextEdge != "" {
		return tm.calculateTurnDirectionFromEdges(currentEdge, nextEdge)
	}

	routeEdges := tm.getVehicleRouteEdges(vehicle.ID)
	if len(routeEdges) > 1 {
		for i, edge := range routeEdges {
			if edge == currentEdge && i < len(routeEdges)-1 {
				return tm.calculateTurnDirectionFromEdges(currentEdge, routeEdges[i+1])
			}
		}
	}

	if strings.Contains(vehicle.Lane, "left") || strings.Contains(vehicle.Lane, "_l") {
		return models.TurnLeft
	} else if strings.Contains(vehicle.Lane, "right") || strings.Contains(vehicle.Lane, "_r") {
		return models.TurnRight
	}

	return models.TurnStraight
}

func (tm *TrafficManager) calculateTurnDirectionFromEdges(currentEdge, nextEdge string) string {
	turnMap := map[string]map[string]string{
		"down_incoming": {
			"left_leaving":  models.TurnRight,
			"down_leaving":  models.TurnStraight,
			"right_leaving": models.TurnLeft,
		},
		"left_incoming": {
			"up_leaving":   models.TurnRight,
			"left_leaving": models.TurnStraight,
			"down_leaving": models.TurnLeft,
		},
		"up_incoming": {
			"right_leaving": models.TurnRight,
			"up_leaving":    models.TurnStraight,
			"left_leaving":  models.TurnLeft,
		},
		"right_incoming": {
			"down_leaving":  models.TurnRight,
			"right_leaving": models.TurnStraight,
			"up_leaving":    models.TurnLeft,
		},
	}

	if edgeMap, exists := turnMap[currentEdge]; exists {
		if direction, hasDirection := edgeMap[nextEdge]; hasDirection {
			return direction
		}
	}

	return models.TurnStraight
}

func (tm *TrafficManager) getVehicleRouteEdges(vehicleID string) []string {
	routeType := ""

	if strings.Contains(vehicleID, "up_to_left") {
		routeType = "from_up_to_left"
	} else if strings.Contains(vehicleID, "up_to_down") {
		routeType = "from_up_to_down"
	} else if strings.Contains(vehicleID, "up_to_right") {
		routeType = "from_up_to_right"
	} else if strings.Contains(vehicleID, "right_to_") {
		routeType = "from_right_to_"
	} else if strings.Contains(vehicleID, "left_to_") {
		routeType = "from_left_to_"
	} else if strings.Contains(vehicleID, "down_to_") {
		routeType = "from_down_to_"
	}

	routes := map[string][]string{
		"from_up_to_left":    {"down_incoming", "left_leaving"},
		"from_up_to_down":    {"down_incoming", "down_leaving"},
		"from_up_to_right":   {"down_incoming", "right_leaving"},
		"from_right_to_left": {"left_incoming", "left_leaving"},
		"from_right_to_up":   {"left_incoming", "up_leaving"},
		"from_right_to_down": {"left_incoming", "down_leaving"},
		"from_left_to_right": {"right_incoming", "right_leaving"},
		"from_left_to_up":    {"right_incoming", "up_leaving"},
		"from_left_to_down":  {"right_incoming", "down_leaving"},
		"from_down_to_up":    {"up_incoming", "up_leaving"},
		"from_down_to_left":  {"up_incoming", "left_leaving"},
		"from_down_to_right": {"up_incoming", "right_leaving"},
	}

	if route, exists := routes[routeType]; exists {
		return route
	}

	return []string{}
}

func (tm *TrafficManager) ManageIntersections() {
	tm.updatePlatoonWaitTimes()
	tm.enforcePlatoonSizeLimits(15)

	for intersectionID, intersection := range tm.Intersections {
		if len(intersection.Vehicles) < 2 {
			continue
		}

		leftTurn := make([]*models.Vehicle, 0)
		rightTurn := make([]*models.Vehicle, 0)
		straight := make([]*models.Vehicle, 0)

		vehiclesByEdge := make(map[string][]*models.Vehicle)
		platoonsByEdge := make(map[string][]string)

		for _, vehicleID := range intersection.Vehicles {
			vehicle, exists := tm.Vehicles[vehicleID]
			if !exists {
				continue
			}

			if vehicle.TurnDirection == "" {
				vehicle.TurnDirection = tm.determineTurnDirection(vehicle, vehicle.NextEdge)
			}

			switch vehicle.TurnDirection {
			case models.TurnLeft:
				leftTurn = append(leftTurn, vehicle)
			case models.TurnRight:
				rightTurn = append(rightTurn, vehicle)
			default:
				straight = append(straight, vehicle)
			}

			edgeKey := vehicle.Edge
			if len(edgeKey) > 0 && edgeKey[0] == ':' {
				if sourceEdge := tm.getSourceEdgeForInternal(vehicle); sourceEdge != "" {
					edgeKey = sourceEdge
				}
			}

			vehiclesByEdge[edgeKey] = append(vehiclesByEdge[edgeKey], vehicle)

			if platoonID, hasPlatoon := tm.VehicleToPlatoon[vehicleID]; hasPlatoon {
				if !tm.containsPlatoon(platoonsByEdge[edgeKey], platoonID) {
					platoonsByEdge[edgeKey] = append(platoonsByEdge[edgeKey], platoonID)
				}
			}
		}

		tm.handleForcedPriorityPlatoons(intersectionID, vehiclesByEdge, platoonsByEdge)
		tm.handleReservations(intersectionID, vehiclesByEdge, platoonsByEdge)
		tm.handlePriorityPlatoons(intersectionID, intersection, vehiclesByEdge, platoonsByEdge)
		tm.handleNonConflictingMovements(intersectionID, vehiclesByEdge, platoonsByEdge, leftTurn, rightTurn, straight)
	}

	tm.handlePostIntersectionVehicles()
}

func (tm *TrafficManager) enforcePlatoonSizeLimits(maxSize int) {
	for platoonID, platoon := range tm.Platoons {
		if len(platoon.VehicleIDs) > maxSize {
			log.Printf("splitting large platoon %s with %d vehicles", platoonID, len(platoon.VehicleIDs))

			sortedVehicles := make([]*models.Vehicle, 0, len(platoon.VehicleIDs))
			for _, vid := range platoon.VehicleIDs {
				if v, exists := tm.Vehicles[vid]; exists {
					sortedVehicles = append(sortedVehicles, v)
				}
			}

			sort.Slice(sortedVehicles, func(i, j int) bool {
				return sortedVehicles[i].Pos > sortedVehicles[j].Pos
			})

			firstGroup := sortedVehicles[:maxSize]
			secondGroup := sortedVehicles[maxSize:]

			if len(secondGroup) > 0 {
				newPlatoonID := fmt.Sprintf("p_split_%s_%d", platoon.Edge, tm.TimeStep)
				newLeader := secondGroup[0]

				newPlatoon := &models.Platoon{
					ID:         newPlatoonID,
					VehicleIDs: make([]string, 0, len(secondGroup)),
					LeaderID:   newLeader.ID,
					Edge:       platoon.Edge,
					Lane:       newLeader.Lane,
				}

				for _, v := range secondGroup {
					newPlatoon.VehicleIDs = append(newPlatoon.VehicleIDs, v.ID)

					v.PlatoonID = newPlatoonID
					v.IsLeader = (v.ID == newLeader.ID)

					tm.VehicleToPlatoon[v.ID] = newPlatoonID
				}

				tm.Platoons[newPlatoonID] = newPlatoon

				platoon.VehicleIDs = make([]string, 0, len(firstGroup))
				for _, v := range firstGroup {
					platoon.VehicleIDs = append(platoon.VehicleIDs, v.ID)
				}
			}
		}
	}
}

func (tm *TrafficManager) updatePlatoonWaitTimes() {
	incomingEdges := map[string]bool{
		"down_incoming":  true,
		"left_incoming":  true,
		"right_incoming": true,
		"up_incoming":    true,
	}

	for _, platoon := range tm.Platoons {
		leader, exists := tm.Vehicles[platoon.LeaderID]
		if !exists {
			continue
		}

		if !incomingEdges[leader.Edge] {
			platoon.IntersectionWaitTime = 0
			platoon.PriorityUntil = nil
			continue
		}

		if leader.Speed < 1.0 && tm.isVehicleAtIntersection(leader) {
			platoon.IntersectionWaitTime += 3

			if len(platoon.VehicleIDs) >= 5 {
				platoon.IntersectionWaitTime += 5
			} else if len(platoon.VehicleIDs) >= 3 {
				platoon.IntersectionWaitTime += 2
			}
		} else {
			platoon.IntersectionWaitTime = 0
		}
	}
}

func (tm *TrafficManager) handleForcedPriorityPlatoons(intersectionID string,
	vehiclesByEdge map[string][]*models.Vehicle, platoonsByEdge map[string][]string) {

	now := time.Now()

	for edge, platoons := range platoonsByEdge {
		for _, platoonID := range platoons {
			platoon, exists := tm.Platoons[platoonID]
			if !exists {
				continue
			}

			if platoon.PriorityUntil != nil && now.Before(*platoon.PriorityUntil) {
				leader, exists := tm.Vehicles[platoon.LeaderID]
				if !exists {
					continue
				}

				log.Printf("MAINTAINING PRIORITY for platoon %s (size: %d, wait: %d) at intersection %s",
					platoonID, len(platoon.VehicleIDs), platoon.IntersectionWaitTime, intersectionID)

				leader.DesiredSpeed = math.Min(leader.Speed+5.0, 19.4)

				for _, vid := range platoon.VehicleIDs {
					if vid == leader.ID {
						continue
					}

					follower, exists := tm.Vehicles[vid]
					if !exists {
						continue
					}

					follower.DesiredSpeed = math.Min(leader.DesiredSpeed, follower.Speed+4.0)
				}

				for otherEdge, vehicles := range vehiclesByEdge {
					if otherEdge == edge {
						continue
					}

					for _, v := range vehicles {
						v.DesiredSpeed = 0.0
					}
				}

				return
			}
		}
	}

	for edge, platoons := range platoonsByEdge {
		for _, platoonID := range platoons {
			platoon, exists := tm.Platoons[platoonID]
			if !exists {
				continue
			}

			leader, exists := tm.Vehicles[platoon.LeaderID]
			if !exists {
				continue
			}

			if len(platoon.VehicleIDs) < 5 && platoon.IntersectionWaitTime < 60 {
				continue
			}

			log.Printf("FORCED PRIORITY for platoon %s (size: %d, wait: %d) at intersection %s",
				platoonID, len(platoon.VehicleIDs), platoon.IntersectionWaitTime, intersectionID)

			priorityDuration := now.Add(15 * time.Second)
			platoon.PriorityUntil = &priorityDuration

			leader.DesiredSpeed = math.Min(leader.Speed+5.0, 19.4)

			for _, vid := range platoon.VehicleIDs {
				if vid == leader.ID {
					continue
				}

				follower, exists := tm.Vehicles[vid]
				if !exists {
					continue
				}

				follower.DesiredSpeed = math.Min(leader.DesiredSpeed, follower.Speed+4.0)
			}

			for otherEdge, vehicles := range vehiclesByEdge {
				if otherEdge == edge {
					continue
				}

				for _, v := range vehicles {
					v.DesiredSpeed = 0.0
				}
			}

			return
		}
	}
}

func (tm *TrafficManager) handlePriorityPlatoons(intersectionID string, intersection *models.Intersection,
	vehiclesByEdge map[string][]*models.Vehicle, platoonsByEdge map[string][]string) {

	now := time.Now()

	if now.Sub(intersection.LastPlatoonPassTime).Seconds() < 3.0 {
		return
	}

	for edge, platoons := range platoonsByEdge {
		for _, platoonID := range platoons {
			platoon, exists := tm.Platoons[platoonID]
			if !exists {
				continue
			}

			if platoon.PriorityUntil != nil && now.Before(*platoon.PriorityUntil) {
				leader, exists := tm.Vehicles[platoon.LeaderID]
				if !exists {
					continue
				}

				log.Printf("MAINTAINING PRIORITY for platoon %s on edge %s (size: %d, wait: %d)",
					platoon.ID, edge, len(platoon.VehicleIDs), platoon.IntersectionWaitTime)

				leader.DesiredSpeed = math.Min(leader.Speed+4.0, 16.7)

				for _, vehicleID := range platoon.VehicleIDs {
					if vehicleID == leader.ID {
						continue
					}

					follower, exists := tm.Vehicles[vehicleID]
					if !exists {
						continue
					}

					follower.DesiredSpeed = math.Min(leader.DesiredSpeed, follower.Speed+3.0)
				}

				for otherEdge, vehicles := range vehiclesByEdge {
					if otherEdge == edge {
						continue
					}

					for _, vehicle := range vehicles {
						otherPlatoonID, inPlatoon := tm.VehicleToPlatoon[vehicle.ID]
						if !inPlatoon {
							vehicle.DesiredSpeed = 0.0
						} else {
							otherPlatoon, exists := tm.Platoons[otherPlatoonID]
							if !exists || otherPlatoon.PriorityUntil == nil ||
								now.After(*otherPlatoon.PriorityUntil) {
								vehicle.DesiredSpeed = 0.0
							}
						}
					}
				}

				intersection.LastPlatoonPassTime = now
				return
			}
		}
	}

	type PlatoonPriority struct {
		platoonID     string
		edge          string
		priorityScore float64
	}

	var priorityQueue []PlatoonPriority

	for edge, platoons := range platoonsByEdge {
		for _, platoonID := range platoons {
			platoon, exists := tm.Platoons[platoonID]
			if !exists {
				continue
			}

			leaderVehicle, exists := tm.Vehicles[platoon.LeaderID]
			if !exists || leaderVehicle.Speed > 3.0 {
				continue
			}

			size := len(platoon.VehicleIDs)
			waitTime := platoon.IntersectionWaitTime

			score := float64(size)*20.0 + float64(waitTime)*10.0

			if size >= 5 {
				score += 150.0
			} else if size >= 3 {
				score += 75.0
			}

			if waitTime > 60 {
				score += 300.0
			} else if waitTime > 30 {
				score += 150.0
			} else if waitTime > 15 {
				score += 75.0
			}

			priorityQueue = append(priorityQueue, PlatoonPriority{
				platoonID:     platoonID,
				edge:          edge,
				priorityScore: score,
			})
		}
	}

	if len(priorityQueue) == 0 {
		return
	}

	sort.Slice(priorityQueue, func(i, j int) bool {
		return priorityQueue[i].priorityScore > priorityQueue[j].priorityScore
	})

	highestPriority := priorityQueue[0]
	platoon, exists := tm.Platoons[highestPriority.platoonID]
	if !exists {
		return
	}

	priorityDuration := now.Add(15 * time.Second)
	platoon.PriorityUntil = &priorityDuration

	leader, exists := tm.Vehicles[platoon.LeaderID]
	if !exists {
		return
	}

	log.Printf("giving priority to platoon %s on edge %s with score %.1f (size: %d, wait: %d)",
		platoon.ID, highestPriority.edge, highestPriority.priorityScore,
		len(platoon.VehicleIDs), platoon.IntersectionWaitTime)

	leader.DesiredSpeed = math.Min(leader.Speed+4.0, 16.7)

	for _, vehicleID := range platoon.VehicleIDs {
		if vehicleID == leader.ID {
			continue
		}

		follower, exists := tm.Vehicles[vehicleID]
		if !exists {
			continue
		}

		follower.DesiredSpeed = math.Min(leader.DesiredSpeed, follower.Speed+3.0)
	}

	for edge, vehicles := range vehiclesByEdge {
		if edge == highestPriority.edge {
			continue
		}

		for _, vehicle := range vehicles {
			otherPlatoonID, inPlatoon := tm.VehicleToPlatoon[vehicle.ID]
			if !inPlatoon {
				vehicle.DesiredSpeed = 0.0
			} else {
				otherPlatoon, exists := tm.Platoons[otherPlatoonID]
				if !exists || otherPlatoon.IntersectionWaitTime < platoon.IntersectionWaitTime/2 {
					vehicle.DesiredSpeed = 0.0
				}
			}
		}
	}

	intersection.LastPlatoonPassTime = now
}

func (tm *TrafficManager) handlePostIntersectionVehicles() {
	postIntersectionEdges := map[string]bool{
		"left_leaving":  true,
		"right_leaving": true,
		"up_leaving":    true,
		"down_leaving":  true,
	}

	vehiclesByLeavingEdge := make(map[string][]*models.Vehicle)

	for _, vehicle := range tm.Vehicles {
		if postIntersectionEdges[vehicle.Edge] && !vehicle.AtIntersection {
			vehiclesByLeavingEdge[vehicle.Edge] = append(vehiclesByLeavingEdge[vehicle.Edge], vehicle)
		}
	}

	for edgeID, vehicles := range vehiclesByLeavingEdge {
		if len(vehicles) < 2 {
			continue
		}

		platoonsByEdge := make(map[string][]*models.Vehicle)

		for _, v := range vehicles {
			platoonID := v.PlatoonID
			if platoonID == "" {
				continue
			}

			platoonsByEdge[platoonID] = append(platoonsByEdge[platoonID], v)
		}

		for platoonID, platoonVehicles := range platoonsByEdge {
			if len(platoonVehicles) < 2 {
				continue
			}

			platoon, exists := tm.Platoons[platoonID]
			if !exists {
				continue
			}

			allOnSameEdge := true
			for _, vid := range platoon.VehicleIDs {
				v, exists := tm.Vehicles[vid]
				if !exists || v.Edge != edgeID {
					allOnSameEdge = false
					break
				}
			}

			if !allOnSameEdge {
				tm.splitPlatoon(platoon, edgeID)
			}
		}

		tm.ensureProperSpacingOnLeavingEdge(edgeID, vehicles)
	}
}

func (tm *TrafficManager) splitPlatoon(platoon *models.Platoon, edgeID string) {
	vehiclesOnEdge := make([]*models.Vehicle, 0)
	vehiclesNotOnEdge := make([]*models.Vehicle, 0)

	for _, vid := range platoon.VehicleIDs {
		v, exists := tm.Vehicles[vid]
		if !exists {
			continue
		}

		if v.Edge == edgeID {
			vehiclesOnEdge = append(vehiclesOnEdge, v)
		} else {
			vehiclesNotOnEdge = append(vehiclesNotOnEdge, v)
		}
	}

	if len(vehiclesOnEdge) < 2 || len(vehiclesNotOnEdge) < 1 {
		return
	}

	newPlatoonID := fmt.Sprintf("p_%s_%d", edgeID, time.Now().UnixNano())

	newLeader := vehiclesOnEdge[0]
	for _, v := range vehiclesOnEdge {
		if v.Pos > newLeader.Pos {
			newLeader = v
		}
	}

	newPlatoon := &models.Platoon{
		ID:         newPlatoonID,
		VehicleIDs: make([]string, 0, len(vehiclesOnEdge)),
		LeaderID:   newLeader.ID,
		Edge:       edgeID,
		Lane:       newLeader.Lane,
	}

	for _, v := range vehiclesOnEdge {
		newPlatoon.VehicleIDs = append(newPlatoon.VehicleIDs, v.ID)

		if v.ID == newLeader.ID {
			v.IsLeader = true
		} else {
			v.IsLeader = false
		}

		v.PlatoonID = newPlatoonID
		tm.VehicleToPlatoon[v.ID] = newPlatoonID

		delete(tm.VehicleToPlatoon, v.ID)
	}

	if len(newPlatoon.VehicleIDs) > 0 {
		tm.Platoons[newPlatoonID] = newPlatoon
	}

	remainingIDs := make([]string, 0, len(vehiclesNotOnEdge))
	for _, v := range vehiclesNotOnEdge {
		remainingIDs = append(remainingIDs, v.ID)
	}

	if len(remainingIDs) < 2 {
		for _, v := range vehiclesNotOnEdge {
			v.PlatoonID = ""
			v.IsLeader = false
			delete(tm.VehicleToPlatoon, v.ID)
		}
		delete(tm.Platoons, platoon.ID)
	} else {
		platoon.VehicleIDs = remainingIDs
	}
}

func (tm *TrafficManager) ensureProperSpacingOnLeavingEdge(edgeID string, vehicles []*models.Vehicle) {
	if len(vehicles) < 2 {
		return
	}

	sortedVehicles := make([]*models.Vehicle, len(vehicles))
	copy(sortedVehicles, vehicles)

	for i := 0; i < len(sortedVehicles)-1; i++ {
		for j := 0; j < len(sortedVehicles)-i-1; j++ {
			if sortedVehicles[j].Pos < sortedVehicles[j+1].Pos {
				sortedVehicles[j], sortedVehicles[j+1] = sortedVehicles[j+1], sortedVehicles[j]
			}
		}
	}

	for i := 1; i < len(sortedVehicles); i++ {
		frontVehicle := sortedVehicles[i-1]
		currentVehicle := sortedVehicles[i]

		gap := frontVehicle.Pos - currentVehicle.Pos

		if gap > 25.0 {
			currentVehicle.DesiredSpeed = 19.4
		} else if gap > 15.0 {
			currentVehicle.DesiredSpeed = math.Min(16.7, frontVehicle.Speed*1.3)
		} else if gap > 10.0 {
			currentVehicle.DesiredSpeed = math.Min(13.9, frontVehicle.Speed*1.2)
		} else if gap < 4.0 {
			currentVehicle.DesiredSpeed = math.Max(5.0, frontVehicle.Speed*0.7)
		} else {
			currentVehicle.DesiredSpeed = frontVehicle.Speed
		}

		if frontVehicle.Speed < 5.0 && gap > 10.0 {
			currentVehicle.DesiredSpeed = 11.1
		}

		if currentVehicle.Speed < 0.5 && gap > 5.0 {
			currentVehicle.DesiredSpeed = 8.3
		}
	}
}

func (tm *TrafficManager) containsPlatoon(platoons []string, platoonID string) bool {
	for _, p := range platoons {
		if p == platoonID {
			return true
		}
	}
	return false
}

func (tm *TrafficManager) handleReservations(intersectionID string,
	vehiclesByEdge map[string][]*models.Vehicle, platoonsByEdge map[string][]string) {

	now := time.Now()

	for _, reservationID := range tm.findReservationsForIntersection(intersectionID) {
		reservation, exists := tm.IntersectionReservations[reservationID]
		if !exists {
			continue
		}

		if now.After(reservation.StartTime) && now.Before(reservation.EndTime) {
			platoon, exists := tm.Platoons[reservation.PlatoonID]
			if !exists {
				continue
			}

			for _, vehicleID := range platoon.VehicleIDs {
				vehicle, exists := tm.Vehicles[vehicleID]
				if !exists || !vehicle.AtIntersection {
					continue
				}

				if vehicle.ID == platoon.LeaderID {
					vehicle.DesiredSpeed = math.Min(vehicle.Speed+3.0, tm.MaxPlatoonSpeed)
				} else {
					leaderVehicle, exists := tm.Vehicles[platoon.LeaderID]
					if !exists {
						continue
					}

					followDist := leaderVehicle.Pos - vehicle.Pos
					if followDist > 20.0 {
						vehicle.DesiredSpeed = math.Min(leaderVehicle.Speed*1.2, leaderVehicle.Speed+5.0)
					} else if followDist < 8.0 {
						vehicle.DesiredSpeed = math.Max(leaderVehicle.Speed*0.8, 5.0)
					} else {
						vehicle.DesiredSpeed = leaderVehicle.Speed
					}
				}

				for otherEdge, otherVehicles := range vehiclesByEdge {
					if otherEdge == reservation.EdgeFrom {
						continue
					}

					for _, otherVehicle := range otherVehicles {
						if tm.areMovementsIncompatible(reservation.EdgeFrom, reservation.Direction,
							otherEdge, otherVehicle.TurnDirection) {
							otherVehicle.DesiredSpeed = math.Max(0.0, otherVehicle.Speed-2.0)
						}
					}
				}
			}
		}
	}
}

func (tm *TrafficManager) areMovementsIncompatible(edge1 string, dir1 string, edge2 string, dir2 string) bool {
	return !tm.areMovementsCompatible(edge1, dir1, edge2, dir2)
}

func (tm *TrafficManager) areMovementsCompatible(edge1 string, dir1 string, edge2 string, dir2 string) bool {
	if edge1 == edge2 {
		return true
	}

	if tm.getOppositeEdge(edge1) == edge2 {
		if dir1 == models.TurnRight && dir2 == models.TurnRight {
			return true
		}
		return false
	}

	if dir1 == models.TurnRight && tm.getLeftEdge(edge2) == edge1 {
		return dir2 != models.TurnStraight && dir2 != models.TurnLeft
	}

	if dir2 == models.TurnRight && tm.getLeftEdge(edge1) == edge2 {
		return dir1 != models.TurnStraight && dir1 != models.TurnLeft
	}

	return false
}

func (tm *TrafficManager) findReservationsForIntersection(intersectionID string) []string {
	var result []string
	for id, reservation := range tm.IntersectionReservations {
		if reservation.IntersectionID == intersectionID {
			result = append(result, id)
		}
	}
	return result
}

func (tm *TrafficManager) getSourceEdgeForInternal(vehicle *models.Vehicle) string {
	parts := strings.Split(vehicle.Edge, "_")
	if len(parts) >= 2 {
		return parts[1]
	}

	return ""
}

func (tm *TrafficManager) handleNonConflictingMovements(
	intersectionID string,
	vehiclesByEdge map[string][]*models.Vehicle,
	platoonsByEdge map[string][]string,
	leftTurn, rightTurn, straight []*models.Vehicle) {

	for _, vehicle := range rightTurn {
		edgeKey := tm.getSourceEdgeForVehicle(vehicle)
		leftEdgeKey := tm.getLeftEdge(edgeKey)

		canProceed := true
		for _, leftVehicle := range vehiclesByEdge[leftEdgeKey] {
			if leftVehicle.TurnDirection == models.TurnStraight {
				canProceed = false
				break
			}
		}

		if canProceed {
			vehicle.DesiredSpeed = math.Min(16.0, vehicle.Speed+3.5)

			platoonID, inPlatoon := tm.VehicleToPlatoon[vehicle.ID]
			if inPlatoon {
				platoon, exists := tm.Platoons[platoonID]
				if exists && platoon.LeaderID == vehicle.ID {
					for _, followerId := range platoon.VehicleIDs {
						if followerId == vehicle.ID {
							continue
						}

						follower, exists := tm.Vehicles[followerId]
						if exists {
							follower.DesiredSpeed = math.Min(14.0, follower.Speed+3.0)
						}
					}
				}
			}

			log.Printf("vehicle %s allowed to turn right at intersection %s",
				vehicle.ID, intersectionID)
		}
	}

	for edgeKey, vehicles := range vehiclesByEdge {
		oppositeEdgeKey := tm.getOppositeEdge(edgeKey)
		oppositeVehicles := vehiclesByEdge[oppositeEdgeKey]

		for _, vehicle := range vehicles {
			if vehicle.TurnDirection != models.TurnLeft {
				continue
			}

			oppositeRightTurns := 0
			oppositeOthers := 0
			for _, oppositeVehicle := range oppositeVehicles {
				if oppositeVehicle.TurnDirection == models.TurnRight {
					oppositeRightTurns++
				} else {
					oppositeOthers++
				}
			}

			if oppositeRightTurns > 0 && oppositeOthers == 0 {
				vehicle.DesiredSpeed = math.Min(10.0, vehicle.Speed+2.0)

				platoonID, inPlatoon := tm.VehicleToPlatoon[vehicle.ID]
				if inPlatoon {
					platoon, exists := tm.Platoons[platoonID]
					if exists && platoon.LeaderID == vehicle.ID {
						for _, followerId := range platoon.VehicleIDs {
							if followerId == vehicle.ID {
								continue
							}

							follower, exists := tm.Vehicles[followerId]
							if exists {
								follower.DesiredSpeed = math.Min(9.0, follower.Speed+1.5)
							}
						}
					}
				}

				log.Printf("vehicle %s allowed to turn left at intersection %s (no conflicts)",
					vehicle.ID, intersectionID)
			}
		}
	}

	northSouthCount := len(vehiclesByEdge["down_incoming"]) + len(vehiclesByEdge["up_incoming"])
	eastWestCount := len(vehiclesByEdge["left_incoming"]) + len(vehiclesByEdge["right_incoming"])

	northSouthPlatoonCount := len(platoonsByEdge["down_incoming"]) + len(platoonsByEdge["up_incoming"])
	eastWestPlatoonCount := len(platoonsByEdge["left_incoming"]) + len(platoonsByEdge["right_incoming"])

	if northSouthPlatoonCount > eastWestPlatoonCount {
		northSouthCount += 5
	} else if eastWestPlatoonCount > northSouthPlatoonCount {
		eastWestCount += 5
	}

	if northSouthCount > eastWestCount+2 {
		for _, edge := range []string{"down_incoming", "up_incoming"} {
			for _, vehicle := range vehiclesByEdge[edge] {
				if vehicle.TurnDirection == models.TurnStraight {
					vehicle.DesiredSpeed = math.Min(14.0, vehicle.Speed+3.0)
				}
			}
		}

		for _, edge := range []string{"left_incoming", "right_incoming"} {
			for _, vehicle := range vehiclesByEdge[edge] {
				if vehicle.TurnDirection == models.TurnStraight {
					vehicle.DesiredSpeed = math.Max(0.0, vehicle.Speed-1.5)
				}
			}
		}
	} else if eastWestCount > northSouthCount+2 {
		for _, edge := range []string{"left_incoming", "right_incoming"} {
			for _, vehicle := range vehiclesByEdge[edge] {
				if vehicle.TurnDirection == models.TurnStraight {
					vehicle.DesiredSpeed = math.Min(14.0, vehicle.Speed+3.0)
				}
			}
		}

		for _, edge := range []string{"down_incoming", "up_incoming"} {
			for _, vehicle := range vehiclesByEdge[edge] {
				if vehicle.TurnDirection == models.TurnStraight {
					vehicle.DesiredSpeed = math.Max(0.0, vehicle.Speed-1.5)
				}
			}
		}
	}
}

func (tm *TrafficManager) getSourceEdgeForVehicle(vehicle *models.Vehicle) string {
	if len(vehicle.Edge) > 0 && vehicle.Edge[0] == ':' {
		return tm.getSourceEdgeForInternal(vehicle)
	}
	return vehicle.Edge
}

func (tm *TrafficManager) getLeftEdge(edge string) string {
	switch edge {
	case "down_incoming":
		return "right_incoming"
	case "left_incoming":
		return "down_incoming"
	case "up_incoming":
		return "left_incoming"
	case "right_incoming":
		return "up_incoming"
	default:
		return ""
	}
}

func (tm *TrafficManager) getOppositeEdge(edge string) string {
	switch edge {
	case "down_incoming":
		return "up_incoming"
	case "left_incoming":
		return "right_incoming"
	case "up_incoming":
		return "down_incoming"
	case "right_incoming":
		return "left_incoming"
	default:
		return ""
	}
}
