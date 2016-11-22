package main

import (
    //"bytes"
    //"fmt"
    "encoding/json"
	//"io/ioutil"
    //"net/http/httputil"
    "log"
    //"net/http"
    "os/exec"
    "time"
    "strings"
)

// getLatestConnectingContainerIDs makes a bulk request to elasticsearch, each individual request being for the latest connections made from a serviceInstance.
// For each response, it edits the corresponding serviceInstance, adding the latest connection to it
func getLatestConnectingContainerIDs() {
    for {
        time.Sleep(5 * time.Second)
        log.Println("getLatestConnectingContainerIDs()")
        //log.Println("getting connections for ", serviceInstance.ContainerID)

        //log.Println(len(serviceInstancesByContainerID), " service instances detected")
        if len(serviceInstancesByContainerID) == 0 {
            log.Println("no service instances found")
            continue
        }

        log.Println("service instances found")

        body := ""
        for _, sInstance := range serviceInstancesByContainerID {
            body += `{}\n{"query":{"term":{"src_addr":"` + sInstance.IPaddress + `"}},"_source":["src_addr","upstream_addr","@timestamp","timestamp_in_ms"],"sort":[{"@timestamp":{"order":"desc"}}]}\n`
        }

        //body += `\n`

        //log.Println(body)
        //if len(body) == 0 {
        //    log.Println("skipping until body length is > 1")
        //    continue
        //}

        //cmdArgs := []string{"echo -e '" + body + "' | ", "-XPOST -H --data-binary @- 'Accept: application/json'","http://localhost:30200/logstash*/_msearch?filter_path=responses.hits.hits._source,responses.hits.hits.sort"}
		//nu, errrr := exec.Command("curl", cmdArgs...).Output()
        nu, errrr := exec.Command("sh", "-c", "echo -e '" + body + "' | curl -XPOST --data-binary @- http://localhost:30200/logstash*/_msearch?filter_path=responses.hits.hits._source,responses.hits.hits.sort").Output()
		if errrr != nil {
			log.Fatal(errrr)
		}
		a := string(nu)
        //log.Println(a)
        var responseBody EsResponse
        json.Unmarshal([]byte(string(a)), &responseBody)

        for _, response := range responseBody.Responses {
            for _, hit := range response.Hits.Hits {
                //if hit.Source != (Source{}) {
                    svcInstance, found := getServiceInstanceByIPAddress(hit.Source.SrcAddr)
                    if found == true {
                        //log.Println("adding adjacency list!")
                        dstInstance, _ := getServiceInstanceByIPAddress(strings.Split(hit.Source.UpstreamAddr, ":")[0])
                        svcInstance.LatestAdjacencyList = []string{dstInstance.ContainerID + ";<container>"}

                        //svcInstance.Edges =  make(map[string]EdgeMetadata)
                        //svcInstance.Edges[dstInstance.ContainerID + ";<container>"] = EdgeMetadata{IngressByteCount:50, EgressByteCount:180}
                        
                        tmp := svcInstance
                        serviceInstancesByContainerID[svcInstance.ContainerID] = tmp
                        log.Println(tmp.ContainerID, " -> ", serviceInstancesByContainerID[tmp.ContainerID].LatestAdjacencyList)//.Edges)
                        //break
                    }
                //}
            }
        }
        
    }
}

type EsResponse struct {
	Responses []struct {
		Hits struct {
			Hits []struct {
				Source struct {
					TimestampInMs string `json:"timestamp_in_ms"`
					Timestamp string `json:"@timestamp"`
					UpstreamAddr string `json:"upstream_addr"`
					SrcAddr string `json:"src_addr"`
				} `json:"_source"`
				Sort []int64 `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	} `json:"responses"`
}