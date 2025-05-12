# VirtualPlatooningIntersectionControl

Bachelors thesis project implementation, where I use SUMO simulator to simulate traffic scenarions by implementing my own Virtual Platooning algorithm and Intersection Control principles. 

Author: Lukáš Lovás

# Quick start

- In the go folder run "go run main.go"
  Optional: --benchmark - turns on benchmark mode that will export statistics into csv every <--duration> steps
            --duration=<Steps>
  Example go run main.go --benchmark --duration=1000
- In your local sumo folder run "sumo-gui --remote-port 1337 -c <path-to-sumo-folder-city.sumocfg>"
- In the python folder run "python main.py"
- localhost:8080 - live statistics interface (!!WORK IN PROGRESS!!)


# V2X-Platooning: Coordinated Intersection Control for Connected Vehicles

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8.svg)](https://go.dev/)
[![Python Version](https://img.shields.io/badge/Python-3.8+-blue.svg)](https://www.python.org/)
[![SUMO Version](https://img.shields.io/badge/SUMO-1.20+-green.svg)](https://www.eclipse.org/sumo/)

V2X-Platooning is a traffic management system that implements Virtual Platooning algorithms to optimize traffic flow at intersections. The system uses vehicle-to-everything (V2X) communication to coordinate the movement of connected vehicles, reducing congestion and improving efficiency.

## Features

- **Virtual Platooning**: Dynamic grouping of vehicles for coordinated intersection traversal
- **Intersection Management**: Priority-based reservation system for intersection crossings
- **Real-Time Simulation**: Integration with SUMO traffic simulator
- **Web Dashboard**: Real-time visualization and control interface
- **Performance Analysis**: Comprehensive statistics collection and benchmarking tools
- **Multiple Intersection Types**: Support for various intersection topologies

## System Architecture

The system consists of three main components:

1. **SUMO Simulator**: Visualizes and simulates the physical movement of vehicles
2. **Python Middleware**: Collects vehicle data from SUMO and communicates with the Traffic Manager
3. **Go Server**: Implements the Virtual Platooning algorithm and handles decision-making

```
┌────────────────┐      ┌─────────────────┐      ┌───────────────┐
│                │      │                 │      │               │
│  SUMO          │◄────►│  Python         │◄────►│  Go Server    │
│  Simulator     │ TraCI│  Middleware     │  TCP │               │
│                │      │                 │      │               │
└────────────────┘      └─────────────────┘      └───────────────┘
```

## 🛠️ Installation

### Prerequisites

- Go 1.24+
- Python 3.8+
- SUMO 1.20+
- TraCI Python library
- Gorilla Websocket Go library

### Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/LukasLovas/VirtualPlatooningIntersectionControl.git
   ```

2. Install Python dependencies:
   ```bash
   pip install traci sumolib
   ```

3. Install Go dependencies: 
   ```bash
   go mod download
   ```
   Note: This will install all dependencies required in go.mod file

4. Build the Go application:
   ```bash
   go build -o traffic-manager
   ```
   or directly run the application by:
    ```bash
    go run main.go
    ```
## 🚗 Running the Simulation

1. Start the SUMO simulator with the TraCI interface:
   ```bash
   sumo-gui -c sumo/city.sumocfg --remote-port 1337
   ```
   Note: from /bin in your SUMO download directory

2. Run the Go Server:
   ```bash
   go run main.go
   ```
   or if previously built:
   ```bash
   go run traffic-manager
   ``` 
3. Start the Python middleware:
   ```bash
   python main.py
   ```

# Main Simulation Loop and Key Methods
The system operates through an iterative simulation loop that processes vehicle data and makes traffic management decisions in real-time:
Simulation Loop Cycle

Data Collection: Python middleware collects vehicle data from SUMO
Data Transmission: Vehicle data is sent to the Go server
State Update: Server updates its internal model of vehicles and platoons
Analysis & Decision: Server runs the Virtual Platooning algorithm
Command Generation: Server creates speed commands for vehicles
Command Transmission: Commands are sent back to the middleware
Command Execution: Middleware applies commands to vehicles in SUMO
Visualization: Current state is displayed in SUMO and the web dashboard

Key Methods
The main processing cycle in the Go server is implemented in the TrafficManager.Update() method, which calls these key methods in each iteration:
## Update(): 
```go
func (tm *TrafficManager) Update() {
	tm.TimeStep++

  tm.UpdatePlatoons()
	tm.EstimatePlatoonStability()
	tm.ReservePlatoonIntersectionSlots()
	tm.ManageIntersections()
	tm.SynchronizeSpeeds()
	tm.AdjustSpeedForTrafficDensity()

	if tm.BenchmarkMode {
		tm.UpdateVehicleThroughput()
		tm.RecordBenchmarkMetrics()
	}
```




`UpdatePlatoons()`

-Updates leader-follower relationships between vehicles
-Forms and updates platoons based on vehicle relationships
-Cleans up empty or invalid platoons
-Checks for transitions between road segments
-Consolidates nearby platoons for optimization

`EstimatePlatoonStability()`

-Analyzes platoon stability based on how long vehicles remain in the same platoon
-Allows higher speeds for stable platoons

`ReservePlatoonIntersectionSlots()`

-Identifies platoons approaching intersections
-Estimates time of arrival at intersections
-Creates time-slot reservations for crossing
-Checks for conflicts with existing reservations

`ManageIntersections()`

-Updates platoon waiting times at intersections
-Assigns priority to platoons with long waiting times
-Processes reservations and checks for conflicts
-Grants priority to platoons based on scoring
-Allows concurrent crossing for non-conflicting trajectories

`SynchronizeSpeeds()`

-Sets vehicle speeds based on distance to the vehicle ahead
-Assigns optimal speed to platoon leaders with reservations
-Synchronizes follower speeds with their leader
-Reduces speed for vehicles without priority

`AdjustSpeedForTrafficDensity()`

-Adjusts speeds based on current traffic density
-Allows higher speeds for larger platoons in lighter traffic
-Dynamically adjusts maximum allowed speeds based on conditions

   
## 🔧 Configuration

The system can be configured using various parameters:

### Vehicle Parameters

- `DetectionDistance`: Maximum distance for vehicle detection (default: 50.0)
- `FollowingGap`: Optimal gap between vehicles in a platoon (default: 10.0)
- `MaxRegularSpeed`: Maximum speed for regular vehicles (default: 15 m/s)
- `MaxPlatoonSpeed`: Maximum speed for platoons (default: 19.4 m/s)
- `StablePlatoonSpeed`: Maximum speed for stable platoons (default: 22.2 m/s)

### Simulation Parameters

- `VEHICLE_INSERT_PROBABILITY`: Probability of inserting a new vehicle (default: 0.3)
- `MAX_VEHICLES`: Maximum number of vehicles in the simulation (default: 30)

## 📊 Benchmarking

The system includes a benchmarking mode to evaluate algorithm performance:

```bash
cd go
go run main.go --benchmark --algorithm=custom --duration=1000
```

Available options:
- `--benchmark`: Enable benchmark mode
- `--algorithm`: Algorithm to use (`custom` meaning custom Virtual platooning implementation or `sumo` for basic Sumo behavior)
- `--duration`: Number of simulation steps

Benchmark results are saved in the `statistics` directory in CSV and JSON formats.

## Intersection Types

The system supports multiple intersection types:

1. **Standard Crossroad (+)**: Four-way intersection (city.net.xml)
2. **Highway with Exits**: Straight road with branches (krizovatka2.net.xml)
3. **Complex Intersection**: Multi-lane intersection with various connections (dialnica.net.xml)

To switch between intersection types, simply use a different SUMO configuration file.

## 📊 Performance Metrics

The system collects the following performance metrics:

- Number of vehicles in the simulation
- Average vehicle speed
- Average waiting time
- Queue size at intersections
- Throughput (vehicles successfully crossing the intersection)
- Number and size of platoons
- Traffic density

## 📁 Project Structure

```
Project/
├── go/                     
│   ├── communication/      # Communication protocol
│   ├── manager/            # Platooning and intersection logic
│   │   ├── benchmark.go    # Performance measurement
│   │   ├── intersection_manager.go # Intersection control
│   │   ├── platoon_operations.go   # Platoon management
│   │   ├── traffic_manager.go      # Main manager
│   │   └── vehicle_operations.go   # Vehicle control
│   ├── models/             # Data structures
│   └── main.go             # Main server module
├── python/                 # Python middleware
│   └── main.py             # TraCI client
├── sumo/                   # SUMO configuration files
│   ├── city.net.xml        # + Intersection network
│   ├── city.rou.xml        # Route definitions
│   ├── city.sumocfg        # SUMO configuration
│   ├── dialnica.net.xml    # Highway with exits network
│   ├── dialnica.rou.xml
│   ├── krizovatka2.net.xml # Simple intersection with branches
│   └── krizovatka2.rou.xml
```

## 📖 Algorithm Description

### Virtual Platooning

The Virtual Platooning algorithm dynamically groups vehicles based on:
1. Leader-follower relationships
2. Physical proximity
3. Lane and road segment sharing
4. Direction of travel

### Platoon Management

Platooning operations include:
- **Formation**: Identifying potential platoons based on vehicle relationships
- **Stability Assessment**: Monitoring platoon stability over time
- **Splitting and Merging**: Dynamic restructuring based on traffic conditions
- **Speed Synchronization**: Coordinating speeds within a platoon

### Intersection Control

The intersection management strategy includes:
- **Reservation System**: Time slot reservation for platoons
- **Priority Assignment**: Based on platoon size and waiting time
- **Conflict Prevention**: Compatibility check of vehicle movements to prevent conflict points
- **Dynamic Speed Adjustment**: Smoothing traffic flow through intersections
