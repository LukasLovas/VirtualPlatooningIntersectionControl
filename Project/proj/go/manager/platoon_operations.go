package manager

import (
	"fmt"
	"math"

	"sumo/models"
)

func (tm *TrafficManager) UpdatePlatoons() {
	tm.updateLeaderRelationships()
	tm.formAndUpdatePlatoons()
	tm.cleanupPlatoons()
	tm.checkEdgeTransitions()
	tm.consolidatePlatoons()
}

func (tm *TrafficManager) updateLeaderRelationships() {
	for _, v := range tm.Vehicles {
		v.LeaderID = ""
	}

	for _, v := range tm.Vehicles {
		leader := tm.FindVehicleAhead(v)
		if leader != nil {
			v.LeaderID = leader.ID
		}
	}
}

func (tm *TrafficManager) formAndUpdatePlatoons() {
	for id, v := range tm.Vehicles {
		if v.LeaderID == "" {
			continue
		}

		leader := tm.Vehicles[v.LeaderID]
		if leader == nil {
			continue
		}

		if leaderPlatoonID, exists := tm.VehicleToPlatoon[leader.ID]; exists {
			if currentPlatoonID, ok := tm.VehicleToPlatoon[id]; !ok || currentPlatoonID != leaderPlatoonID {
				if ok && currentPlatoonID != leaderPlatoonID {
					tm.RemoveVehicleFromPlatoon(id, currentPlatoonID)
				}

				tm.AddVehicleToPlatoon(id, leaderPlatoonID)
			}
		} else {
			gap := leader.Pos - v.Pos

			if gap <= 25.0 && !isEdgeTransition(leader.Edge, v.Edge) {
				platoonID := fmt.Sprintf("p_%s_%s", leader.Edge, leader.ID)

				tm.Platoons[platoonID] = &models.Platoon{
					ID:         platoonID,
					VehicleIDs: []string{leader.ID, id},
					LeaderID:   leader.ID,
					Edge:       leader.Edge,
					Lane:       leader.Lane,
				}

				tm.VehicleToPlatoon[leader.ID] = platoonID
				tm.VehicleToPlatoon[id] = platoonID

				leader.PlatoonID = platoonID
				leader.IsLeader = true
				v.PlatoonID = platoonID
				v.IsLeader = false
			}
		}
	}
}

func isEdgeTransition(edge1, edge2 string) bool {
	isLeavingEdge := func(edge string) bool {
		switch edge {
		case "left_leaving", "right_leaving", "up_leaving", "down_leaving":
			return true
		default:
			return false
		}
	}

	isIncomingEdge := func(edge string) bool {
		switch edge {
		case "left_incoming", "right_incoming", "up_incoming", "down_incoming":
			return true
		default:
			return false
		}
	}

	if isLeavingEdge(edge1) && isIncomingEdge(edge2) {
		return true
	}

	if isIncomingEdge(edge1) && isLeavingEdge(edge2) {
		return true
	}

	return false
}

func (tm *TrafficManager) cleanupPlatoons() {
	for platoonID, platoon := range tm.Platoons {
		if len(platoon.VehicleIDs) <= 1 {
			for _, vid := range platoon.VehicleIDs {
				if vehicle, exists := tm.Vehicles[vid]; exists {
					vehicle.PlatoonID = ""
					vehicle.IsLeader = false
				}
				delete(tm.VehicleToPlatoon, vid)
			}

			delete(tm.Platoons, platoonID)
		}
	}
}

func (tm *TrafficManager) checkEdgeTransitions() {
	leavingEdges := map[string]bool{
		"left_leaving":  true,
		"right_leaving": true,
		"up_leaving":    true,
		"down_leaving":  true,
	}

	incomingEdges := map[string]bool{
		"left_incoming":  true,
		"right_incoming": true,
		"up_incoming":    true,
		"down_incoming":  true,
	}

	for platoonID, platoon := range tm.Platoons {
		vehiclesByEdge := make(map[string][]*models.Vehicle)

		for _, vid := range platoon.VehicleIDs {
			v, exists := tm.Vehicles[vid]
			if !exists {
				continue
			}

			vehiclesByEdge[v.Edge] = append(vehiclesByEdge[v.Edge], v)
		}

		if len(vehiclesByEdge) <= 1 {
			continue
		}

		splitNeeded := false

		for edge1, _ := range vehiclesByEdge {
			if !leavingEdges[edge1] && !incomingEdges[edge1] {
				continue
			}

			for edge2, _ := range vehiclesByEdge {
				if edge1 == edge2 {
					continue
				}

				if (leavingEdges[edge1] && incomingEdges[edge2]) ||
					(leavingEdges[edge2] && incomingEdges[edge1]) {
					splitNeeded = true
					break
				}
			}

			if splitNeeded {
				break
			}
		}

		if splitNeeded {
			for edge, edgeVehicles := range vehiclesByEdge {
				if len(edgeVehicles) >= 2 {
					if leavingEdges[edge] {
						tm.createNewPlatoonFromVehicles(edgeVehicles, edge)
					}
				}
			}

			delete(tm.Platoons, platoonID)

			for _, vid := range platoon.VehicleIDs {
				if _, exists := tm.VehicleToPlatoon[vid]; exists {
					delete(tm.VehicleToPlatoon, vid)
				}
			}
		}
	}
}

func (tm *TrafficManager) createNewPlatoonFromVehicles(vehicles []*models.Vehicle, edge string) {
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

	newLeader := sortedVehicles[0]
	newPlatoonID := fmt.Sprintf("p_%s_%s_%d", edge, newLeader.ID, tm.TimeStep)

	newPlatoon := &models.Platoon{
		ID:         newPlatoonID,
		VehicleIDs: make([]string, 0, len(sortedVehicles)),
		LeaderID:   newLeader.ID,
		Edge:       edge,
		Lane:       newLeader.Lane,
	}

	for _, v := range sortedVehicles {
		newPlatoon.VehicleIDs = append(newPlatoon.VehicleIDs, v.ID)

		v.PlatoonID = newPlatoonID
		v.IsLeader = (v.ID == newLeader.ID)

		tm.VehicleToPlatoon[v.ID] = newPlatoonID
	}

	tm.Platoons[newPlatoonID] = newPlatoon
}

func (tm *TrafficManager) consolidatePlatoons() {
	for _, platoon := range tm.Platoons {
		leader, exists := tm.Vehicles[platoon.LeaderID]
		if !exists {
			continue
		}

		for _, otherID := range tm.FindNearbyPlatoonLeaders(leader) {
			otherLeader, exists := tm.Vehicles[otherID]
			if !exists || otherLeader.Edge != leader.Edge || otherLeader.Lane != leader.Lane {
				continue
			}

			distance := math.Abs(otherLeader.Pos - leader.Pos)
			if distance > 30.0 {
				continue
			}

			otherPlatoonID, hasPlatoon := tm.VehicleToPlatoon[otherID]
			if !hasPlatoon {
				continue
			}

			otherPlatoon, exists := tm.Platoons[otherPlatoonID]
			if !exists || otherPlatoon.ID == platoon.ID {
				continue
			}

			if otherLeader.Pos > leader.Pos {
				tm.mergePlatoons(platoon.ID, otherPlatoon.ID)
			} else {
				tm.mergePlatoons(otherPlatoon.ID, platoon.ID)
			}
		}
	}
}

func (tm *TrafficManager) FindNearbyPlatoonLeaders(vehicle *models.Vehicle) []string {
	result := make([]string, 0)

	for id, v := range tm.Vehicles {
		if id == vehicle.ID {
			continue
		}

		if !v.IsLeader || v.Edge != vehicle.Edge || v.Lane != vehicle.Lane {
			continue
		}

		distance := math.Abs(v.Pos - vehicle.Pos)
		if distance <= 30.0 {
			result = append(result, id)
		}
	}

	return result
}

func (tm *TrafficManager) mergePlatoons(leadingPlatoonID, trailingPlatoonID string) {
	leadingPlatoon, exists1 := tm.Platoons[leadingPlatoonID]
	trailingPlatoon, exists2 := tm.Platoons[trailingPlatoonID]

	if !exists1 || !exists2 {
		return
	}

	newVehicleIDs := make([]string, 0, len(leadingPlatoon.VehicleIDs)+len(trailingPlatoon.VehicleIDs))
	newVehicleIDs = append(newVehicleIDs, leadingPlatoon.VehicleIDs...)

	for _, vid := range trailingPlatoon.VehicleIDs {
		if !tm.containsVehicle(leadingPlatoon.VehicleIDs, vid) {
			newVehicleIDs = append(newVehicleIDs, vid)
		}

		vehicle, exists := tm.Vehicles[vid]
		if exists && vid != trailingPlatoon.LeaderID {
			vehicle.PlatoonID = leadingPlatoonID
			tm.VehicleToPlatoon[vid] = leadingPlatoonID
		}
	}

	trailingLeader, exists := tm.Vehicles[trailingPlatoon.LeaderID]
	if exists {
		trailingLeader.IsLeader = false
		trailingLeader.PlatoonID = leadingPlatoonID
		tm.VehicleToPlatoon[trailingLeader.ID] = leadingPlatoonID
	}

	leadingPlatoon.VehicleIDs = newVehicleIDs

	delete(tm.Platoons, trailingPlatoonID)
}

func (tm *TrafficManager) containsVehicle(vehicles []string, vehicleID string) bool {
	for _, id := range vehicles {
		if id == vehicleID {
			return true
		}
	}
	return false
}

func (tm *TrafficManager) AddVehicleToPlatoon(vehicleID, platoonID string) {
	platoon, exists := tm.Platoons[platoonID]
	if !exists {
		return
	}

	vehicle, exists := tm.Vehicles[vehicleID]
	if !exists {
		return
	}

	if !tm.containsVehicle(platoon.VehicleIDs, vehicleID) {
		platoon.VehicleIDs = append(platoon.VehicleIDs, vehicleID)
	}

	vehicle.PlatoonID = platoonID
	vehicle.IsLeader = false

	tm.VehicleToPlatoon[vehicleID] = platoonID
}

func (tm *TrafficManager) RemoveVehicleFromPlatoon(vehicleID, platoonID string) {
	platoon, exists := tm.Platoons[platoonID]
	if !exists {
		return
	}

	for i, vid := range platoon.VehicleIDs {
		if vid == vehicleID {
			platoon.VehicleIDs = append(platoon.VehicleIDs[:i], platoon.VehicleIDs[i+1:]...)
			break
		}
	}

	if vehicleID == platoon.LeaderID && len(platoon.VehicleIDs) > 0 {
		newLeaderID := platoon.VehicleIDs[0]
		platoon.LeaderID = newLeaderID
		if newLeader, exists := tm.Vehicles[newLeaderID]; exists {
			newLeader.IsLeader = true
		}
	}

	if vehicle, exists := tm.Vehicles[vehicleID]; exists {
		vehicle.PlatoonID = ""
		vehicle.IsLeader = false
	}
}

func (tm *TrafficManager) GetPlatoonsForVisualization() map[string]map[string]interface{} {
	platoons := make(map[string]map[string]interface{})

	for id, platoon := range tm.Platoons {
		platoonData := map[string]interface{}{
			"leader":   platoon.LeaderID,
			"vehicles": platoon.VehicleIDs,
			"edge":     platoon.Edge,
			"lane":     platoon.Lane,
		}
		platoons[id] = platoonData
	}

	return platoons
}
