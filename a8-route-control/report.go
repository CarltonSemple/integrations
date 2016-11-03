package main

import (
    "encoding/json"
    "log"
    "net/http"
    "time"
)

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

func (p *Plugin) makeReport() (*report, error) {
	// Add the container IDs to the map of nodes in the report
	m := make(map[string]node)
	log.Println("making ", p.ID, " report")

	if len(serviceInstancesByContainerID) == 0 {
		log.Println("no services found")
	}
	for _, v := range serviceInstancesByContainerID {
		
		key := v.ContainerID + ";<container>"
		switch {
			case p.ID == "a8routing": {
				metrics, weightValue, err := p.routingPercentage(v)
				if metrics == nil {
					//log.Println("skipping")
					continue
				}
				if err != nil {
					return nil, err
				}
				v.Weight = weightValue
				//log.Println(v.Weight)
				p.routesEnabled = true
				m[key] = node { 
					Metrics:        metrics,
					LatestControls: p.latestControls(),
					AdjacencyList: []string{""},
					Rank: "8",
				}
			}
			case p.ID == "a8connections": {
				//log.Println("desired list:")
				//log.Println(v.DesiredAdjacencyList)
				if p.ConnectionsType == "actual" {
					m[key] = node { 
						LatestControls: p.latestControls(),
						AdjacencyList: v.LatestAdjacencyList,
						Rank: "8",
					}
				} else {
					m[key] = node { 
						LatestControls: p.latestControls(),
						AdjacencyList: v.DesiredAdjacencyList,
						Rank: "8",
					}
				}
			}
		}
	}
	rpt := &report{
		Container: topology{
			Nodes: m,
			MetricTemplates: p.metricTemplates(),
			Controls:        p.controls(),
			//TableTemplates: getTableTemplate(),
		},
		Plugins: []pluginSpec{
			{
				ID:          p.ID,//"a8routing",
				Label:       p.Label,//"a8routing",
				Description: p.Description,//"Adds routing to different versions of a microservice",
				Interfaces:  []string{"reporter", "controller"},
				APIVersion:  "1",
			},
		},
	}

	return rpt, nil
}

func (p *Plugin) latestControls() map[string]controlEntry {
	ts := time.Now()
	ctrls := map[string]controlEntry{}
	for _, details := range p.allControlDetails() {
		ctrls[details.id] = controlEntry{
			Timestamp: ts,
			Value: controlData{
				Dead: details.dead,
			},
		}
	}
	return ctrls
}

func (p *Plugin) metricTemplates() map[string]metricTemplate {
	id, name := p.routingMetricIDAndName()
	return map[string]metricTemplate{
		id: {
			ID:       id,
			Label:    name,
			DataType: "",//Format:   "percent",
			Priority: 0.1,//13.5,//
		},
	}
}

func getTableTemplate() map[string]tableTemplate {
	return map[string]tableTemplate{
		"a8routing-table": {
			ID:     "a8routing-table",
			Label:  "Amalgam8 Routing Control",
			Prefix: "a8routing-table-",
		},
	}
}

func (p *Plugin) routingMetricIDAndName() (string, string) {
	return "routeamount", "Routing Weight"
}

// Define the controls in the topology report @ /report
func (p *Plugin) controls() map[string]control {
	ctrls := map[string]control{}
	for _, details := range p.allControlDetails() {
		ctrls[details.id] = control{
			ID:    details.id,
			Human: details.human,
			Icon:  details.icon,
			Rank:  1,
		}
	}
	return ctrls
}

// Report is called by scope when a new report is needed. It is part of the
// "reporter" interface, which all plugins must implement.
func (p *Plugin) Report(w http.ResponseWriter, r *http.Request) {
	p.lock.Lock()
	defer p.lock.Unlock()
	log.Println(r.URL.String())
	rpt, err := p.makeReport()
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