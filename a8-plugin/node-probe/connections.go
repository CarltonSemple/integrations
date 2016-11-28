package main

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "os/exec"
    "strings"
    "time"
)

type ContainerConnection struct {
    SourceIP string `json:sourceIP`
    SourceDockerID string `json:sourceDockerID`
    DestinationIP string `json:destinationIP`
    DestinationDockerID string `json:destinationDockerID`
}

type ConnectionsHolder struct {
    connections []ContainerConnection `json:containers`
}

func getLatestA8ContainerConnectionInformation(containersByIP map[string]ContainerSimple) (map[string][]ContainerConnection, map[string]serviceInstance, error) {
    var connectionsByIPAddress = make(map[string][]ContainerConnection)
    a8ServiceInstances, err := getAmalgam8ServiceInstancesByIPAddress()
    if err != nil {
        return connectionsByIPAddress, a8ServiceInstances, fmt.Errorf("getLatestA8ContainerConnectionInformation: ", err)
    }
    body := ""
    for _, container := range containersByIP {
        ipAddress := container.IP
        body += `{}\n{"query":{"term":{"src_addr":"` + ipAddress + `"}},"_source":["src_addr","upstream_addr","@timestamp","timestamp_in_ms"],"sort":[{"@timestamp":{"order":"desc"}}]}\n`
    }

    nu, errrr := exec.Command("sh", "-c", "echo -e '" + body + "' | curl -XPOST --data-binary @- " + os.Getenv("ES_URL") + "/logstash*/_msearch?filter_path=responses.hits.hits._source,responses.hits.hits.sort").Output()
    if errrr != nil {
        return connectionsByIPAddress, a8ServiceInstances, errrr
    }
    a := string(nu)
    var responseBody EsResponse
    json.Unmarshal([]byte(string(a)), &responseBody)

    for _, response := range responseBody.Responses {
        for _, hit := range response.Hits.Hits {
            if svcInstance, found := a8ServiceInstances[hit.Source.SrcAddr]; found {
                destinationInstance, ok := a8ServiceInstances[strings.Split(hit.Source.UpstreamAddr, ":")[0]]
                if ok == false {
                    log.Println("destinationInstance: ", ok)
                    continue
                }
                // Skip if the connection is too long ago
                occurenceTimeMillis := hit.Sort[0] / 1000
                occurenceTime := time.Unix(occurenceTimeMillis, 0)                
                now := time.Now()
                diff := now.Sub(occurenceTime)
                if diff.Minutes() > connectionTimeWindow_minutes {
                    log.Println("connection too old; skipping")
                    continue
                }
                
                // Add what we know about the connection
                newConnection := ContainerConnection{}
                if sourceContainer, isFound := containersByIP[svcInstance.GetIPAddress()]; isFound {
                    newConnection.SourceDockerID = sourceContainer.ID
                }
                if destinationContainer, isFound := containersByIP[destinationInstance.GetIPAddress()]; isFound {
                    newConnection.DestinationDockerID = destinationContainer.ID
                }
                newConnection.SourceIP = svcInstance.GetIPAddress()
                newConnection.DestinationIP = destinationInstance.GetIPAddress()

                connectionsByIPAddress[newConnection.SourceIP] = append(connectionsByIPAddress[newConnection.SourceIP], newConnection)
                break
            }
        }
    }

    return connectionsByIPAddress, a8ServiceInstances, nil
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