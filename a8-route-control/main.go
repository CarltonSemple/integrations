package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

// Icons can be found at http://fontawesome.io/icons/

const dependencyServerPort = ":3000"

func main() {
	serviceInstancesByContainerID = make(map[string]serviceInstance)
	desiredAdjacencyListsByServiceName = make(map[string][]string)
	latestRouteReport = &report{}
	latestConnectionReport = &report{}
	go hostDockerQuery()
	go getAmalgam8ContainerIds()

	hostname, _ := os.Hostname()
	var (
		routingAddress   = flag.String("routingAddress", "/var/run/scope/plugins/a8routing.sock", "unix socket to listen for connections on")
		connectionsAddress   = flag.String("connectionsAddress", "/var/run/scope/plugins/a8connections.sock", "unix socket to listen for connections on")
		hostID = flag.String("hostname", hostname, "hostname of the host running this plugin")
	)
	flag.Parse()

	log.Println(routingAddress)
	log.Println(connectionsAddress)
	log.Println(hostID)

	go mainRoutingPlugin(routingAddress, hostID)
	go mainConnectionsPlugin(connectionsAddress, hostID, true)
	go userConnectionDependenciesServer()

	for {
		log.Println("HELLO FROM MAIN")
		time.Sleep(5 * time.Second)
	}
}

func userConnectionDependenciesServer() {
	server := http.NewServeMux()
	// sample::::::: curl localhost:3000/submit -d '{"gremlins":[{"scenario":"delay_requests","source":"productpage:v1","dest":"ratings:v1","delaytime":"7s"}]}'
	server.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.String())

		grmBody := GremlinContainer{}
		err := json.NewDecoder(r.Body).Decode(&grmBody)
		if err != nil {
			log.Printf("Bad request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Println("gremlineeee-----------------------------------------")
		log.Println(grmBody.Gremlins[0])

		for _, grem := range grmBody.Gremlins {
			for key, instance := range serviceInstancesByContainerID {
				if instance.Name == strings.Split(grem.Source, ":")[0] && listContains(instance.Tags, strings.Split(grem.Source, ":")[1]) {
					log.Println("***********************************************desired adjacency found for ", instance.Name)
					tmp := instance
					iiis := getServiceInstancesByName(strings.Split(grem.Dest, ":")[0], strings.Split(grem.Dest, ":")[1])
					var insContainerIds []string
					for _, ins := range iiis {
						insContainerIds = append(insContainerIds, ins.ContainerID + ";<container>")
					}
					desiredAdjacencyListsByServiceName[tmp.Name] = insContainerIds //append(desiredAdjacencyListsByServiceName[tmp.Name], insContainerIds)//strings.Split(grem.Dest, ":")[0])
					tmp.DesiredAdjacencyList = desiredAdjacencyListsByServiceName[tmp.Name]
					serviceInstancesByContainerID[key] = tmp
					log.Println(serviceInstancesByContainerID[key].DesiredAdjacencyList)
					break
				}
			}
			log.Println("************************************************next gremlin")
		}

		raw, _ := json.Marshal(grmBody)
		w.WriteHeader(http.StatusOK)
		w.Write(raw)
	})
	if err := http.ListenAndServe(dependencyServerPort, server); err != nil {
		log.Printf("error: %v", err)
	}
}

func mainConnectionsPlugin(addr *string, hostID *string, actualConnections bool) {
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

	plugin := &Plugin{ActualConnections: actualConnections, HostID: *hostID, ID: "a8connections", Label: "a8connections", Description: "Shows connections between microservices"}
	go buildReports(plugin)
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
	go buildReports(plugin)
	server := http.NewServeMux()
	server.HandleFunc("/report", plugin.Report)
	server.HandleFunc("/control", plugin.Control)
	if err := http.Serve(listener, server); err != nil {
		log.Printf("error: %v", err)
	}
}

func buildReports(p *Plugin) {
	for {
		p.buildLatestReport()
		time.Sleep(2 * time.Second)
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

	ActualConnections bool
}

type request struct {
	NodeID  string
	Control string
}

type response struct {
	ShortcutReport *report `json:"shortcutReport,omitempty"`
}

type GremlinContainer struct {
	Gremlins []Gremlin `json:"gremlins"`
}

type Gremlin struct {
	Scenario string `json:"scenario"`
	Source string `json:"source"`
	Dest string `json:"dest"`
	Delaytime string `json:"delaytime"`
}
