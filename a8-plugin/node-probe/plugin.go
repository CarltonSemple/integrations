package main

import (
	"sync"
	"time"
)

// Plugin groups the methods a plugin needs
type Plugin struct {
	HostID string

	lock       sync.Mutex
	routesEnabled bool

	LatestReport *report

	ID string
	Label string
	Description string

	ActualConnections bool
	Containers []ContainerSimple
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
