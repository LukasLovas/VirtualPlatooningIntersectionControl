# VirtualPlatooningIntersectionControl

My own bachelors thesis project implementation, where I use SUMO simulator to simulate traffic scenarions by implementing my own Virtual Platooning algorithm and Intersection Control principles. 

Author: Lukáš Lovás

- In the go folder run "go run main.go"
  Optional: --benchmark - turns on benchmark mode that will export statistics into csv every <--duration> steps
            --duration=<Steps>
  Example go run main.go --benchmark --duration=1000
- In your local sumo folder run "sumo-gui --remote-port 1337 -c <path-to-sumo-folder-city.sumocfg>"
- In the python folder run "python main.py"
- localhost:8080 - live statistics interface (!!WORK IN PROGRESS!!)
