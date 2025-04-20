package models

import "time"

type Vehicle struct {
	ID                  string    `json:"id"`
	Lane                string    `json:"lane"`
	Pos                 float64   `json:"pos"`
	Speed               float64   `json:"speed"`
	Edge                string    `json:"edge"`
	PlatoonID           string    `json:"-"`
	IsLeader            bool      `json:"-"`
	DesiredSpeed        float64   `json:"-"`
	LeaderID            string    `json:"-"`
	NextEdge            string    `json:"-"`
	TurnDirection       string    `json:"-"`
	AtIntersection      bool      `json:"-"`
	LastSpeedChange     time.Time `json:"-"`
	StablePlatoonTime   float64   `json:"-"`
	ReactionTime        float64   `json:"-"`
	WaitingTime         int       `json:"-"`
	CountedInThroughput bool      `json:"-"`
	CreationTime        time.Time `json:"-"`
	TravelTime          float64   `json:"-"`
}

type EdgeStatistics struct {
	VehicleCount       int
	PlatoonCount       int
	LargestPlatoonSize int
	MaxWaitTime        int
	TotalWaitTime      int
	PriorityScore      float64
}

type IntersectionControlState struct {
	CurrentPriorityEdge string
	PriorityStartTime   time.Time
	MinGreenTime        int
	MaxGreenTime        int
}

type Platoon struct {
	ID                   string
	VehicleIDs           []string
	LeaderID             string
	Edge                 string
	Lane                 string
	StabilityRatio       float64
	IntersectionWaitTime int
	PriorityUntil        *time.Time
}

type Intersection struct {
	ID                  string
	Edges               []string
	InternalID          string
	Vehicles            []string
	HasReservation      bool
	LastPlatoonPassTime time.Time
	CurrentControlState *IntersectionControlState
}

type IntersectionReservation struct {
	ID             string
	IntersectionID string
	PlatoonID      string
	StartTime      time.Time
	EndTime        time.Time
	EdgeFrom       string
	Direction      string
}

const (
	TurnLeft     = "left"
	TurnRight    = "right"
	TurnStraight = "straight"
)
