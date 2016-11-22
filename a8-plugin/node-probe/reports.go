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
    containerTag = ";<container>"
)

func buildConnectionReports(connectionsPlugin *Plugin) {
    log.Println("buildConnectionReports")
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
    log.Println("starting connections loop")
    for {
        err := connectionsPlugin.generateLatestActualConnectionsReport(clientset)
        if err != nil {
            log.Println("generateLatestActualConnectionsReport: ", err)
        } 
        time.Sleep(5 * time.Second)
    }
}

func (p *Plugin) generateLatestActualConnectionsReport(clientset *kubernetes.Clientset) error {
    log.Println("generateLatestActualConnectionsReport")
    m := make(map[string]node)
    containers, err := GetContainersFromKubernetesGoClient(clientset)
    if err != nil {
        log.Println("GetContainersFromKubernetes: ", err)
    }
    filteredContainers := filterOutSidecarContainers(containers...)
    log.Println("filtered containers:")
    log.Println(filteredContainers)
    var containersByIPAddress map[string]ContainerSimple
    containers, containersByIPAddress, err = filterToAmalgam8Containers(filteredContainers...)
    if err != nil {
        log.Println("1: ", err)
        return err
    }
    connectionsByIPAddress, serviceInstances, _ := getLatestA8ContainerConnectionInformation(containersByIPAddress)
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

    log.Println("iterating through serviceInstances")

    for _, instance := range serviceInstances {
        serviceIPaddress := instance.GetIPAddress()
        // attach to each container with this ip address
        connections, ok := connectionsByIPAddress[serviceIPaddress]
        if ok {
            log.Println(len(connections))
            instance.LatestAdjacencyList = []string{}
            for _, connection := range connections {
                key := connection.SourceDockerID + containerTag
                log.Println("key:", key)
                //instance.toLatestAdjacencyList([]string{connection.destinationDockerID})
                instance.LatestAdjacencyList = append(instance.LatestAdjacencyList, connection.DestinationDockerID + containerTag)
                if len(instance.LatestAdjacencyList) > 0 {
                    log.Println("adjacency list:")
                    log.Println(instance.LatestAdjacencyList)
                }
                m[key] = node { 
                    AdjacencyList: instance.LatestAdjacencyList,
                    Rank: "8",
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
			DataType: "",
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

func filterOutSidecarContainers(containers ...ContainerSimple) []ContainerSimple {
    newContainers := []ContainerSimple{}
    for _, c := range containers{
        if c.Name != "servicereg" && c.Name != "serviceproxy" {
            newContainers = append(newContainers, c)
        }
    }
    return newContainers
}

type report struct {
	Container    topology
	Plugins []pluginSpec
}

type topology struct {
	Nodes           map[string]node           `json:"nodes"`
	MetricTemplates map[string]metricTemplate `json:"metric_templates"`//`json:"metadata_templates,omitempty"`//
	Controls        map[string]control        `json:"controls"`
	TableTemplates 	map[string]tableTemplate  `json:"table_templates,omitempty"`
}

type tableTemplate struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Prefix string `json:"prefix"`
}

type node struct {
	Metrics        map[string]metric       `json:"metrics"`
	LatestControls map[string]controlEntry `json:"latestControls,omitempty"`
	AdjacencyList []string `json:"adjacency",omitempty`
	Edges map[string]EdgeMetadata `json:"edges,omitempty"`
	Rank string `json:rank,omitempty`
}

type metric struct {
	Samples []sample `json:"samples,omitempty"`
	Min     float64  `json:"min"`
	Max     float64  `json:"max"`
}

type sample struct {
	Date  time.Time `json:"date"`
	Value float64   `json:"value"`
}

type controlEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Value     controlData `json:"value"`
}

type controlData struct {
	Dead bool `json:"dead"`
}

type metricTemplate struct {
	ID       string  `json:"id"`
	Label    string  `json:"label,omitempty"`
	DataType string  `json:"dataType,omitempty"`
	Format   string  `json:"format,omitempty"`
	Priority float64 `json:"priority,omitempty"`
}

type control struct {
	ID    string `json:"id"`
	Human string `json:"human"`
	Icon  string `json:"icon"`
	Rank  int    `json:"rank"`
}

type pluginSpec struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Interfaces  []string `json:"interfaces"`
	APIVersion  string   `json:"api_version,omitempty"`
}
