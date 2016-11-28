package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
    "k8s.io/client-go/1.5/kubernetes"
    "k8s.io/client-go/1.5/rest"
)

const (
    containerChannelSize = 3
)
var containersChannel = make(chan []ContainerSimple, containerChannelSize)

func buildConnectionReports(connectionsPlugin *Plugin) {
    log.Println("buildConnectionReports")
    connectionsPlugin.LatestReport = &report{
        Container: topology{
            MetricTemplates: connectionsPlugin.metricTemplates(),
        },
        Plugins: []pluginSpec{
            {
                ID:          "a8connections",
                Label:       "a8connections",
                Description:  "Shows connections between microservices",
                Interfaces:  []string{"reporter"},
                APIVersion:  "1",
            },
        },
    }
    for {
        err := connectionsPlugin.generateLatestActualConnectionsReport()
        if err != nil {
            log.Println("generateLatestActualConnectionsReport: ", err)
        }
        time.Sleep(5 * time.Second)
    }
}

func updateContainersCollection() {
    config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
    for {
        containers, err := GetContainersFromKubernetesGoClient(clientset)
        if err != nil {
            log.Println("updateContainersCollection: ", err)
        }
        if len(containersChannel) == containerChannelSize {
            <- containersChannel
        }
        containersChannel <- containers
        time.Sleep(containerRefreshRate_seconds * time.Second)
    }
}

func (p *Plugin) generateLatestActualConnectionsReport() error {
    log.Println("generateLatestActualConnectionsReport")
    m := make(map[string]node)
    if len(containersChannel) > 0 {
        p.Containers = <- containersChannel
    }
    //filteredContainers := filterOutSidecarContainers(p.Containers...)
    //// Moved to filterToAmalgam8Containers()
    _, containersByIPAddress, err := filterToAmalgam8Containers(p.Containers...)
    if err != nil {
        log.Println("1: ", err)
        return err
    }
    connectionsByIPAddress, serviceInstances, err := getLatestA8ContainerConnectionInformation(containersByIPAddress)
    if err != nil {
        return err
    }
    _, err = json.Marshal(connectionsByIPAddress)
    if err != nil {
        log.Println("2: ", err)
        return err
    }
    log.Println(connectionsByIPAddress)
    if len(serviceInstances) == 0 {
        log.Println("no services found")
        return nil
    }

    for _, instance := range serviceInstances {
        serviceIPaddress := instance.GetIPAddress()
        // attach to each container with this ip address
        connections, ok := connectionsByIPAddress[serviceIPaddress]
        if ok {
            //log.Println(len(connections))
            instance.LatestAdjacencyList = []string{}
            for _, connection := range connections {
                key := connection.SourceDockerID + containerTag
                log.Println("key:", key)
                instance.LatestAdjacencyList = append(instance.LatestAdjacencyList, connection.DestinationDockerID + containerTag)
                if len(instance.LatestAdjacencyList) > 0 {
                    log.Println("adjacency list:")
                    log.Println(instance.LatestAdjacencyList)
                }
                m[key] = node { 
                    AdjacencyList: instance.LatestAdjacencyList,
                    //Rank: "8",
                }
            }
        }
    }

    p.LatestReport = &report{
        Container: topology{
            Nodes: m,
            MetricTemplates: p.metricTemplates(),
        },
        Plugins: []pluginSpec{
            {
                ID:          "a8connections",
                Label:       "a8connections",
                Description:  "Shows connections between microservices",
                Interfaces:  []string{"reporter"},
                APIVersion:  "1",
            },
        },
    }

    return nil
}

func (p *Plugin) getLatestReport() (*report, error) {
    if p.LatestReport == (&report{}) {
        return p.LatestReport, fmt.Errorf("latest report is empty")
    }
    return p.LatestReport, nil
}

func (p *Plugin) metricTemplates() map[string]metricTemplate {
	id, name := "routeamount", "Routing Weight"
    if p.ID == "a8connections" {
        id = "connections"
        name = "Connections"
    }
	return map[string]metricTemplate{
		id: {
			ID:       id,
			Label:    name,
			DataType: "",//Format:   "percent",
			Priority: 0.1,
		},
	}
}

// Report is called by scope when a new report is needed. It is part of the
// "reporter" interface, which all plugins must implement.
func (p *Plugin) Report(w http.ResponseWriter, r *http.Request) {
	p.lock.Lock()
	defer p.lock.Unlock()
	log.Println(r.URL.String())
	rpt, err := p.getLatestReport()
	if err != nil {
		log.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	raw, err := json.Marshal(*rpt)
	if err != nil {
		log.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//log.Println(string(raw))
	w.WriteHeader(http.StatusOK)
	w.Write(raw)
}
