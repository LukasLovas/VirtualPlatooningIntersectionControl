# VirtualPlatooningIntersectionControl

My own bachelors thesis project implementation, where I use SUMO simulator to simulate traffic scenarions by implementing my own Virtual Platooning algorithm and Intersection Control principles. 

Author: Lukáš Lovás

#Quick start

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
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

V2X-Platooning is a traffic management system that implements Virtual Platooning algorithms to optimize traffic flow at intersections. The system uses vehicle-to-everything (V2X) communication to coordinate the movement of connected vehicles, reducing congestion and improving efficiency.

## 🚀 Features

- **Virtual Platooning**: Dynamic grouping of vehicles for coordinated intersection traversal
- **Intersection Management**: Priority-based reservation system for intersection crossings
- **Real-Time Simulation**: Integration with SUMO traffic simulator
- **Web Dashboard**: Real-time visualization and control interface
- **Performance Analysis**: Comprehensive statistics collection and benchmarking tools
- **Multiple Intersection Types**: Support for various intersection topologies

## 📋 System Architecture

The system consists of three main components:

1. **SUMO Simulator**: Visualizes and simulates the physical movement of vehicles
2. **Python Middleware**: Collects vehicle data from SUMO and communicates with the Traffic Manager
3. **Go Traffic Manager**: Implements the Virtual Platooning algorithm and handles decision-making

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

### Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/v2x-platooning.git
   cd v2x-platooning
   ```

2. Install Python dependencies:
   ```bash
   pip install traci sumolib
   ```

3. Build the Go application:
   ```bash
   cd go
   go build -o traffic-manager
   ```

## 🚗 Running the Simulation

1. Start the SUMO simulator with the TraCI interface:
   ```bash
   sumo-gui -c sumo/city.sumocfg --remote-port 1337
   ```

2. Run the Go Traffic Manager:
   ```bash
   cd go
   ./traffic-manager
   ```
   Or directly with Go:
   ```bash
   go run main.go
   ```

3. Start the Python middleware:
   ```bash
   cd python
   python main.py
   ```

4. Access the web dashboard at: http://localhost:8080 (Work in progress)

## 🔧 Configuration

The system can be configured using various parameters:

### Vehicle Parameters

- `DetectionDistance`: Maximum distance for vehicle detection (default: 50.0)
- `FollowingGap`: Optimal gap between vehicles in a platoon (default: 10.0)
- `MaxRegularSpeed`: Maximum speed for regular vehicles (default: 16.7 m/s)
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
- `--algorithm`: Algorithm to use (`custom` or `sumo`)
- `--duration`: Number of simulation steps

Benchmark results are saved in the `statistics` directory in CSV and JSON formats.

## 🌟 Intersection Types

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
│   └── main.go             # Entry point
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
