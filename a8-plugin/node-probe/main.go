package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	//"sync"
	"time"
)

// TODO: remove
var ClusterIP = os.Getenv("KUBERNETES_SERVICE_HOST") + ":" + os.Getenv("KUBERNETES_SERVICE_PORT")
var latestConnectionReport = &report{}
var connectionTimeWindow_minutes = 15.0

const (
	containerTag = ";<container>"
	containerRefreshRate_seconds = 10
)

func main() {
	hostname, _ := os.Hostname()
	var (
		routingAddress   = flag.String("routingAddress", "/var/run/scope/plugins/a8routing.sock", "unix socket to listen for connections on")
		connectionsAddress   = flag.String("connectionsAddress", "/var/run/scope/plugins/a8connections.sock", "unix socket to listen for connections on")
		hostID = flag.String("hostname", hostname, "hostname of the host running this plugin")
	)
	flag.Parse()
	log.Println("A8Plugin running on host ", hostID)

	go updateContainersCollection()
	
	go mainConnectionsPlugin(connectionsAddress, hostID, true)
	go mainRoutingPlugin(routingAddress, hostID)
	for {
		log.Println("hello from Main")
		time.Sleep(15 * time.Second)
	}
}


func mainConnectionsPlugin(addr *string, hostID *string, actualConnections bool) {
	log.Printf("Connections Plugin starting on %s...\n", *hostID)

	os.Remove(*addr)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		os.Remove(*addr)
		os.Exit(0)
	}()

	listener, err := net.Listen("unix", *addr)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		listener.Close()
		os.Remove(*addr)
	}()

	log.Printf("Connections Listening on: unix://%s", *addr)

	plugin := &Plugin{ActualConnections: actualConnections, HostID: *hostID, ID: "a8connections", Label: "a8connections", Description: "Shows connections between microservices"}
	
	go buildConnectionReports(plugin)
	
	server := http.NewServeMux()
	server.HandleFunc("/report", plugin.Report)
	if err := http.Serve(listener, server); err != nil {
		log.Printf("error: %v", err)
	}
}

func mainRoutingPlugin(addr *string, hostID *string) {
	log.Printf("Routing Plugin starting on %s...\n", *hostID)

	os.Remove(*addr)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		os.Remove(*addr)
		os.Exit(0)
	}()

	listener, err := net.Listen("unix", *addr)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		listener.Close()
		os.Remove(*addr)
	}()

	log.Printf("Listening on: unix://%s", *addr)

	plugin := &Plugin{HostID: *hostID, ID: "a8routing", Label: "a8routing", Description: "Adds routing to different versions of a microservice"}
	
	go buildRouteControlReports(plugin)

	server := http.NewServeMux()
	server.HandleFunc("/report", plugin.Report)
	server.HandleFunc("/control", plugin.Control)
	if err := http.Serve(listener, server); err != nil {
		log.Printf("error: %v", err)
	}
}