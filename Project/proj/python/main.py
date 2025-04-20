import traci
import json
import socket
import time
import random

HOST = "localhost"
PORT = 5555

platoon_colors = {}

VEHICLE_INSERT_PROBABILITY = 0.5
MAX_VEHICLES = 50
VEHICLE_TYPES = ["car"]
MIN_DEPARTURE_POSITION = 0.0
MAX_DEPARTURE_POSITION = 5.0

def send_to_go(sock, vehicle_states):
    msg = json.dumps(vehicle_states).encode()
    sock.sendall(len(msg).to_bytes(4, "big") + msg)

def recv_from_go(sock):
    raw_len = sock.recv(4)
    if not raw_len:
        return None
    msg_len = int.from_bytes(raw_len, "big")
    data = b""
    while len(data) < msg_len:
        part = sock.recv(msg_len - len(data))
        if not part:
            break
        data += part
    return json.loads(data.decode())

def gather_vehicle_data():
    vehicle_ids = traci.vehicle.getIDList()
    data = {}
    for vid in vehicle_ids:
        try:
            data[vid] = {
                "lane": traci.vehicle.getLaneID(vid),
                "pos": traci.vehicle.getLanePosition(vid),
                "speed": traci.vehicle.getSpeed(vid),
                "edge": traci.vehicle.getRoadID(vid),
            }
        except traci.TraCIException:
            continue
    return data

def initialize_vehicle_types():
    try:
        traci.vehicletype.copy("DEFAULT_VEHTYPE", "car")
        traci.vehicletype.setLength("car", 5.0)
        traci.vehicletype.setWidth("car", 2.0)
        traci.vehicletype.setHeight("car", 1.5)
        traci.vehicletype.setMaxSpeed("car", 50.0)
        traci.vehicletype.setAccel("car", 2.5)
        traci.vehicletype.setDecel("car", 4.5)
        print("vehicle type 'car' created successfully")
    except traci.TraCIException as e:
        print(f"error initializing vehicle types: {e}")

def add_vehicles():
    if len(traci.vehicle.getIDList()) >= MAX_VEHICLES:
        return
        
    try:
        routes = traci.route.getIDList()
        if not routes:
            print("no routes available in simulation!")
            return
    except traci.TraCIException as e:
        print(f"error getting routes: {e}")
        return
        
    if random.random() < VEHICLE_INSERT_PROBABILITY:
        route_id = random.choice(routes)
        
        veh_id = f"veh_{int(time.time())}_{random.randint(0, 10000)}"
        
        try:
            traci.vehicle.add(
                veh_id,
                route_id,
                typeID="car",
                departLane="0",
                departPos="0",
                departSpeed="0"
            )
            
            traci.vehicle.setMaxSpeed(veh_id, random.uniform(10.0, 15.0))
            traci.vehicle.setColor(veh_id, (255, 255, 255, 255))
            
            print(f"added vehicle {veh_id} on route {route_id}")
            
        except traci.TraCIException as e:
            print(f"failed to add vehicle: {e}")

def clean_departed_vehicles():
    arrived = traci.simulation.getArrivedIDList()
    if arrived:
        print(f"vehicles completed routes: {', '.join(arrived)}")

def apply_commands(cmds):
    if not cmds:
        return
    
    if "speeds" in cmds:
        for vid, speed in cmds["speeds"].items():
            try:
                if vid in traci.vehicle.getIDList():
                    traci.vehicle.setSpeed(vid, float(speed))
            except traci.TraCIException as e:
                print(f"error setting speed for {vid}: {e}")
    
    if "platoons" in cmds:
        global platoon_colors
        
        default_colors = [
            (255, 0, 0, 255),
            (0, 255, 0, 255),
            (0, 0, 255, 255),
            (255, 255, 0, 255),
            (255, 0, 255, 255),
            (0, 255, 255, 255),
            (128, 0, 0, 255),
            (0, 128, 0, 255),
            (0, 0, 128, 255),
        ]
        
        colored_vehicles = set()
        
        for platoon_id, platoon_data in cmds["platoons"].items():
            if platoon_id not in platoon_colors:
                color_idx = len(platoon_colors) % len(default_colors)
                platoon_colors[platoon_id] = default_colors[color_idx]
            
            color = platoon_colors[platoon_id]
            
            leader_id = platoon_data["leader"]
            if leader_id in traci.vehicle.getIDList():
                leader_color = tuple(min(255, c * 1.3) for c in color[:3]) + (255,)
                traci.vehicle.setColor(leader_id, leader_color)
                traci.vehicle.setWidth(leader_id, 2.2)
                colored_vehicles.add(leader_id)
            
            for vid in platoon_data["vehicles"]:
                if vid != leader_id and vid in traci.vehicle.getIDList():
                    traci.vehicle.setColor(vid, color)
                    traci.vehicle.setWidth(vid, 2.0)
                    colored_vehicles.add(vid)
        
        for vid in traci.vehicle.getIDList():
            if vid not in colored_vehicles:
                traci.vehicle.setColor(vid, (255, 255, 255, 255))
                traci.vehicle.setWidth(vid, 1.8)
        
        existing_platoons = set(cmds["platoons"].keys())
        for platoon_id in list(platoon_colors.keys()):
            if platoon_id not in existing_platoons:
                del platoon_colors[platoon_id]
    
    if "stats" in cmds:
        stats = cmds["stats"]
        print(f"Step {stats['time_step']}: {stats['vehicle_count']} vehicles, "
              f"{stats['platoon_count']} platoons")

def main():
    traci.init(port=1337)
    print("connected to SUMO via TraCI.")
    
    initialize_vehicle_types()
    
    routes = traci.route.getIDList()
    print(f"available routes: {routes}")

    with socket.create_connection((HOST, PORT)) as sock:
        print("connected to Go traffic manager.")
        
        step = 0
        try:
            while True:
                add_vehicles()
                
                traci.simulationStep()
                
                clean_departed_vehicles()
                
                vehicle_data = gather_vehicle_data()
                
                send_to_go(sock, vehicle_data)
                
                cmds = recv_from_go(sock)
                if cmds:
                    apply_commands(cmds)
                
                if step % 10 == 0:
                    print(f"simulation step {step}, {len(traci.vehicle.getIDList())} vehicles active")
                
                step += 1
                
                time.sleep(0.05)
                
        except KeyboardInterrupt:
            print("stopping simulation.")
        finally:
            traci.close()

if __name__ == "__main__":
    main()