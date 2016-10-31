package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

// Icons can be found at http://fontawesome.io/icons/

func main() {
	serviceInstancesByContainerID = make(map[string]serviceInstance)
	go hostDockerQuery()
	go getAmalgam8ContainerIds()

	hostname, _ := os.Hostname()
	var (
		routingAddress   = flag.String("routingAddress", "/var/run/scope/plugins/a8routing.sock", "unix socket to listen for connections on")
		connectionsAddress   = flag.String("connectionsAddress", "/var/run/scope/plugins/a8connections.sock", "unix socket to listen for connections on")
		hostID = flag.String("hostname", hostname, "hostname of the host running this plugin")
	)
	flag.Parse()

	log.Println(connectionsAddress)

	go mainRoutingPlugin(routingAddress, hostID)
	go mainConnectionsPlugin(connectionsAddress, hostID)

	for {
		log.Println("HELLO FROM MAIN")
		time.Sleep(5 * time.Second)
	}
}

func mainConnectionsPlugin(addr *string, hostID *string) {
	log.Printf("Connections Plugin starting on %s...\n", *hostID)

	go getLatestConnectingContainerIDs()

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

	plugin := &Plugin{HostID: *hostID, ID: "a8connections", Label: "a8connections", Description: "Shows connections between microservices"}
	server := http.NewServeMux()
	server.HandleFunc("/report", plugin.Report)
	server.HandleFunc("/control", plugin.Control)
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
	server := http.NewServeMux()
	server.HandleFunc("/report", plugin.Report)
	server.HandleFunc("/control", plugin.Control)
	if err := http.Serve(listener, server); err != nil {
		log.Printf("error: %v", err)
	}
}

// Plugin groups the methods a plugin needs
type Plugin struct {
	HostID string

	lock       sync.Mutex
	routesEnabled bool

	ID string
	Label string
	Description string
}

type request struct {
	NodeID  string
	Control string
}

type response struct {
	ShortcutReport *report `json:"shortcutReport,omitempty"`
}