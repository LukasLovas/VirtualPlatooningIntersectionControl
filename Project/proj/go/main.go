package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	network "sumo/communication"
	"sumo/manager"
	"sumo/web"
)

func main() {
	benchmarkMode := flag.Bool("benchmark", false, "Run in benchmark mode")
	algorithmType := flag.String("algorithm", "custom", "Traffic algorithm to use (custom or sumo)")
	duration := flag.Int("duration", 1000, "Benchmark duration in steps")
	flag.Parse()

	tm := manager.NewTrafficManager()
	tm.UseCustomAlgorithm = (*algorithmType == "custom")

	os.MkdirAll("statistics", 0755)
	os.MkdirAll("web/static/css", 0755)
	os.MkdirAll("web/static/js", 0755)
	os.MkdirAll("web/templates", 0755)

	webServer := web.NewWebServer(tm)
	go webServer.Start()
	log.Printf("Web interface started on http://localhost:8080")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		log.Println("shutdown signal received, exiting...")
		os.Exit(0)
	}()

	listener, err := net.Listen("tcp", "localhost:5555")
	if err != nil {
		log.Fatalf("failed to listen on port 5555: %v", err)
	}
	defer listener.Close()

	log.Printf("traffic manager waiting for Python client on port 5555...")

	conn, err := listener.Accept()
	if err != nil {
		log.Fatalf("filed to accept connection: %v", err)
	}
	defer conn.Close()

	webServer.SetSumoConnection(conn)
	log.Printf("connection established with Python client and linked to web server")

	if *benchmarkMode {
		tm.StartBenchmark(*duration, *algorithmType)
	}

	for {
		vehicleData, err := network.ReceiveVehicleData(conn)
		if err != nil {
			log.Printf("err receiving data: %v", err)
			break
		}

		tm.UpdateVehicleData(vehicleData)
		tm.Update()

		commands := tm.PrepareCommands()
		err = network.SendCommands(conn, commands)
		if err != nil {
			log.Printf("err sending commands: %v", err)
			break
		}

		time.Sleep(10 * time.Millisecond)
	}
}
