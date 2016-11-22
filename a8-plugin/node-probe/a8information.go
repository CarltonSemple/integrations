package main

import (
    "encoding/json"
    "log"
	"os"
    "os/exec"
    "strings"
)

// get one of the containers' docker ID at this ip address
//func getContainerIDsfromIP(ipAddress string, connectionsByIP map[string]ContainerConnection) []string {

//}

func filterToAmalgam8Containers(containers ...ContainerSimple) ([]ContainerSimple, map[string]ContainerSimple, error){
	log.Println("filterToAmalgam8Containers()")

	containerMapByIPAddress := make(map[string]ContainerSimple)
	var newContainerList []ContainerSimple
	amalgam8ServiceInstances := getAmalgam8ServiceInstancesByIPAddress()

	for _, container := range containers {
		ipAddress := container.IP

		// Check to see if this container is in the collection of Amalgam8 services
		if _, ok := amalgam8ServiceInstances[ipAddress]; ok { 
			newContainerList = append(newContainerList, container)
			containerMapByIPAddress[ipAddress] = container
		}
	}
	return newContainerList, containerMapByIPAddress, nil
}

func getAmalgam8ServiceInstancesByIPAddress() map[string]serviceInstance {
	//log.Println("getAmalgam8ServiceInstancesByIPAddress")
	m := make(map[string]serviceInstance) // IP addresses are the keys to the map of instances
	cmdArgs := []string{"-H 'Accept: application/json'",os.Getenv("A8_REGISTRY_URL") + "/api/v1/services"}
	o, errrr := exec.Command("curl", cmdArgs...).Output()
	if errrr != nil {
		log.Println("no services received")
	}
	s := string(o)
	var svcResponse serviceListResponse
	json.Unmarshal([]byte(s), &svcResponse)
	
	// Get the IP address of each service
	for _, serviceName := range svcResponse.Services {	
		foundInstances := GetServiceInstances(serviceName)

		// Add each instance of the service to the list
		for _, instance := range foundInstances {
			ip := strings.Split(instance.Endpoint.Value, ":")
			instance.IPaddress = ip[0]
			m[ip[0]] = instance
		}
	}
	return m
}

// GetServiceInstances returns 
func GetServiceInstances(serviceName string) []serviceInstance {
	var svcDetails serviceDetails
	cmdArgs := []string{"-H 'Accept: application/json'",os.Getenv("A8_REGISTRY_URL") + "/api/v1/services/" + serviceName}
	nu, errrr := exec.Command("curl", cmdArgs...).Output()
	if errrr != nil {
		log.Fatal(errrr)
	}
	a := string(nu)
	json.Unmarshal([]byte(a), &svcDetails)
	return svcDetails.Instances
}



/**********
*
* Structs for handling the Amalgam8 service list API responses
*
***********/


type EdgeMetadata struct {
	EgressPacketCount  uint64 `json:"egress_packet_count,omitempty"`
	IngressPacketCount uint64 `json:"ingress_packet_count,omitempty"`
	EgressByteCount    uint64 `json:"egress_byte_count,omitempty"`  // Transport layer
	IngressByteCount   uint64 `json:"ingress_byte_count,omitempty"` // Transport layer
}

// Value is typically the IP address of the service instance
type serviceEndpoint struct {
	Type string `json:type`
	Value string `json:value`
}

type serviceInstance struct {
	Id string `json:"id"`
	Name string `json:"service_name"`
	Endpoint serviceEndpoint `json:endpoint`
	Tags []string `json:tags`
	ContainerID string `json:"containerid,omitempty"`
	IPaddress string `json:"ip,omitempty"`
	LatestAdjacencyList []string `json:"adjacencyList,omitempty"` //`json:"adjacencyList,omitempty"`
	DesiredAdjacencyList []string `json:"adjacencyListDesired,omitempty"`
	Edges map[string]EdgeMetadata `json:"edges,omitempty"`
	Weight float64
}

type serviceDetails struct {
	Name string `json:"service_name"`
	Instances []serviceInstance `json:instances`
}

type serviceListResponse struct {
	Services []string `json:services`
}

func (svc *serviceInstance) GetIPAddress() string {
	if svc.Endpoint != (serviceEndpoint{}) {
		return strings.Split(svc.Endpoint.Value, ":")[0]
	}
	return ""
}

func (svc *serviceInstance) toLatestAdjacencyList(connections []ContainerConnection) {
	latestList := []string{}
	for _, connection := range connections {
		latestList = append(latestList, connection.SourceDockerID + ";<container>")
	}
}

/*
var desiredAdjacencyListsByServiceName map[string][]string

// Look at the Amalgam8 IP addresses and use this list to filter the list of ID/IP pairs from hostDockerQuery
func getAmalgam8ContainerIds() (map[string]serviceInstance, error) {
	log.Println("getAmalgam8ContainerIds")

	// Get the IP addresses of Amalgam8 containers
	//addressMap := findAmalgam8Addresses()
	m := make(map[string]serviceInstance) // containerIDs are the keys to this map of instances

	containerList, err := getContainerList()
	if err != nil {
		return m, err
	}

	// map of service instances by IP address
	serviceInstances := getAmalgam8ServiceInstancesByIPAddress()

	// Add the pairs with Amalgam8 IP addresses to our collection
	for _, container := range containerList {
		ipAddress, err := container.GetIPAddress()
		if err != nil {
			return m, err
		}
		// Check to see if this container is in the collection of Amalgam8 services
		if _, ok := serviceInstances[ipAddress]; ok { 
			var tmp = serviceInstances[ipAddress]
			tmp.ContainerID = container.ID
			tmp.IPaddress = ipAddress

			// update desired adjacency list
			tmp.DesiredAdjacencyList = desiredAdjacencyListsByServiceName[tmp.Name]
			
			serviceInstances[ipAddress] = tmp
			m[container.ID] = tmp
		}
	}
	return m
}

// getServiceInstanceByAddress looks through the map of service instances (with container IDs as keys) and returns the instance with the desired IP address
// returns empty, false if not found
func getServiceInstanceByIPAddress(ipAddress string) (serviceInstance, bool) {
	//log.Println("getServiceInstanceByAddress")
	for _, sInstance := range serviceInstancesByContainerID {
		if sInstance.IPaddress == ipAddress {
			//log.Println("matched ", sInstance.IPaddress)
			return sInstance, true
		}
	}
	return serviceInstance{}, false
}

// getServiceInstancesByName looks through the map of service instances and returns the ones that have the same name
func getServiceInstancesByName(name string, versionTag string) []serviceInstance {
	var instances []serviceInstance
	for _, sInstance := range serviceInstancesByContainerID {
		if sInstance.Name == name && listContains(sInstance.Tags, versionTag) {
			instances = append(instances, sInstance)
		}
	}
	return instances
}

// GetServiceVersionIPAddress returns the IP address from /api/v1/services/serviceName, along with the port
func GetServiceVersionIPAddress(serviceName string, versionTag string) (string, string) {
	foundInstances := GetServiceInstances(serviceName)
	for _, instance := range foundInstances {
		if listContains(instance.Tags, versionTag) == true {
			ipport := strings.Split(instance.Endpoint.Value, ":")
			return ipport[0], ipport[1]
		}
	}
	return "", ""
}

func listContains(stringList []string, sInQuestion string) bool {
	for _, x := range stringList {
		if x == sInQuestion {
			return true
		}
	}
	return false
}*/