package main

import (
    //"encoding/json"
    "log"
    //"net/http"
    "time"
)

var latestRouteReport *report

func buildRouteControlReports(routePlugin *Plugin) {
    log.Println("buildRouteControlReports")
    routePlugin.LatestReport = &report{
        Container: topology{
            MetricTemplates: routePlugin.metricTemplates(),
            Controls:        routePlugin.controls(),
        },
        Plugins: []pluginSpec{
            {
                ID:          "a8routing",
                Label:       "a8routing",
                Description: "Adds routing to different versions of a microservice",
                Interfaces:  []string{"reporter", "controller"},
                APIVersion:  "1",
            },
        },
    }
    for {
        err := routePlugin.generateLatestRouteControlReport()
        if err != nil {
            log.Println("generateLatestRouteControlReport: ", err)
        }
        time.Sleep(3 * time.Second)
    }
}

func (p *Plugin) generateLatestRouteControlReport() error {
    log.Println("generateLatestRouteControlReport()")
	m := make(map[string]node)
    if len(containersChannel) > 0 {
        p.Containers = <- containersChannel
    }
    containers, _, err := filterToAmalgam8Containers(p.Containers...)
    if err != nil {
        log.Println("1: ", err)
        return err
    }
    serviceInstancesByContainerID, err := getAmalgam8ServiceInstancesByContainerID(containers...)
    if len(serviceInstancesByContainerID) == 0{
        log.Println("no services found")
        return nil
    }
    if err != nil {
        return err
    }
	for _, v := range serviceInstancesByContainerID {		
		key := v.ContainerID + containerTag
        metrics, weightValue, _ := p.routingPercentage(v)
        if metrics == nil {
            continue
        }
        v.Weight = weightValue
        p.routesEnabled = true
        m[key] = node { 
            Metrics:        metrics,
            LatestControls: p.latestControls(),
            Rank: "8",
        }
	}

    p.LatestReport = &report{
        Container: topology{
            Nodes: m,
            MetricTemplates: p.metricTemplates(),
            Controls:        p.controls(),
            //TableTemplates: getTableTemplate(),
        },
        Plugins: []pluginSpec{
            {
                ID:          "a8routing",
                Label:       "a8routing",
                Description: "Adds routing to different versions of a microservice",
                Interfaces:  []string{"reporter", "controller"},
                APIVersion:  "1",
            },
        },
    }
    return nil
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

func getTableTemplate() map[string]tableTemplate {
	return map[string]tableTemplate{
		"a8routing-table": {
			ID:     "a8routing-table",
			Label:  "Amalgam8 Routing Control",
			Prefix: "a8routing-table-",
		},
	}
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