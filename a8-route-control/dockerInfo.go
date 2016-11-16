package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "net/http"
)

const (
    dockerSocketPath = "/var/run/docker.sock"  
)

func getContainerList() ([]Container, error) {
    var receivedContainers []Container

    tr := &http.Transport{
        Dial: unixSocketDial,
    }
    client := &http.Client{Transport: tr}
    resp, err := client.Get("http://localhost/containers/json")
    if err != nil {
        log.Println(err)
        return receivedContainers, err
    }
    rbody, _ := ioutil.ReadAll(resp.Body)
    json.Unmarshal([]byte(rbody), &receivedContainers)

    return receivedContainers, nil
}

func unixSocketDial(proto, addr string) (conn net.Conn, err error) {
    return net.Dial("unix", dockerSocketPath)
}

type Container struct {
	ID string `json:"Id"`
	Names []string `json:"Names"`
	Image string `json:"Image"`
	ImageID string `json:"ImageID"`
	Command string `json:"Command"`
	Created int `json:"Created"`
	Ports []interface{} `json:"Ports"`
	Labels interface {} `json:"Labels"`
	State string `json:"State"`
	Status string `json:"Status"`
	HostConfig struct {
		NetworkMode string `json:"NetworkMode"`
	} `json:"HostConfig"`
	NetworkSettings struct {
		Networks struct {
			Host Network `json:"host"`
            Bridge Network `json:"bridge"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
	Mounts []struct {
		Source string `json:"Source"`
		Destination string `json:"Destination"`
		Mode string `json:"Mode"`
		RW bool `json:"RW"`
		Propagation string `json:"Propagation"`
	} `json:"Mounts"`
}

type Network struct {
    IPAMConfig interface{} `json:"IPAMConfig"`
    Links interface{} `json:"Links"`
    Aliases interface{} `json:"Aliases"`
    NetworkID string `json:"NetworkID"`
    EndpointID string `json:"EndpointID"`
    Gateway string `json:"Gateway"`
    IPAddress string `json:"IPAddress"`
    IPPrefixLen int `json:"IPPrefixLen"`
    IPv6Gateway string `json:"IPv6Gateway"`
    GlobalIPv6Address string `json:"GlobalIPv6Address"`
    GlobalIPv6PrefixLen int `json:"GlobalIPv6PrefixLen"`
    MacAddress string `json:"MacAddress"`
}

func (c *Container) GetIPAddress() (string, error) {
    if c.NetworkSettings.Networks.Bridge != (Network{}) {
        return c.NetworkSettings.Networks.Bridge.IPAddress, nil
    }
    return "", fmt.Errorf("Bridge Network does not exist for Container %s", c.ID)
}