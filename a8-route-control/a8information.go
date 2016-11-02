package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "log"
    "net"
    "os/exec"
    "strings"
    "time"
)

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



type idAddressPair struct {
	ID string `json:id`
	IP string `json:ip`
}

var latestHostServerResponse string

// hostDockerQuery queries the server running on the host for a list of 
// running Container IDs (docker ps) paired with IP addresses
func hostDockerQuery() {
	log.Println("hostDockerQuery")
	for {
		time.Sleep(2 * time.Second)
		c, err := net.Dial("unix", "/var/run/dockerConnection/hostconnection.sock")
		if err != nil {
			continue;
		}
		// send to socket
		log.Println("sending request to server")
		fmt.Fprintf(c, "hi" + "\n")
		// listen for reply
		message, _ := bufio.NewReader(c).ReadString('\n')
		//log.Println("Message from server: " + message)
		log.Println("Received update from host server")

		// set  this to be the latest response
		latestHostServerResponse = message
	}
} 

// map of service instances, with the IP addresses as the keys
//var serviceInstances []serviceInstance
// map of service instances, with the container ID as the key
var serviceInstancesByContainerID map[string]serviceInstance

func updateAmalgam8ServiceInstances() map[string]serviceInstance{
	log.Println("updateAmalgam8ServiceInstances")
	m := make(map[string]serviceInstance) // IP addresses are the keys to the map of instances

	// amalgam8
	cmdArgs := []string{"-H 'Accept: application/json'","http://localhost:31300/api/v1/services"}
	o, errrr := exec.Command("curl", cmdArgs...).Output()
	if errrr != nil {
		log.Println("no services received")
		//log.Fatal(errrr)
	}
	s := string(o)
	var svcResponse serviceListResponse
	json.Unmarshal([]byte(s), &svcResponse)
	
	// Get the IP address of each service
	for _, serviceName := range svcResponse.Services {
		/*var svcDetails serviceDetails
		//log.Println(serviceName)
		cmdArgs = []string{"-H 'Accept: application/json'","http://localhost:31300/api/v1/services/" + serviceName}
		nu, errrr := exec.Command("curl", cmdArgs...).Output()
		if errrr != nil {
			log.Fatal(errrr)
		}
		a := string(nu)
		json.Unmarshal([]byte(a), &svcDetails)*/
		
		foundInstances := GetServiceInstances(serviceName)

		// Add each instance of the service to the list
		//for _, instance := range svcDetails.Instances {
		for _, instance := range foundInstances {
			ip := strings.Split(instance.Endpoint.Value, ":")
			m[ip[0]] = instance
		}
	}
	log.Println("Updated Amalgam8 Service Instances without Container IDs")
	return m
}

func getAllContainerIdAddressPairs(serverJsonString string) []idAddressPair {
	log.Println("getAllContainerIdAddressPairs")
	var pairs []idAddressPair = make([]idAddressPair, 0)
	if len(serverJsonString) == 0 {
		return pairs
	}
	json.Unmarshal([]byte(serverJsonString), &pairs)
	return pairs
}

// Look at the Amalgam8 IP addresses and use this list to filter the list of ID/IP pairs from hostDockerQuery
func getAmalgam8ContainerIds() {
	for {
		log.Println("getAmalgam8ContainerIds")
		time.Sleep(3 * time.Second)

		// Get the IP addresses of Amalgam8 containers
		//addressMap := findAmalgam8Addresses()
		m := make(map[string]serviceInstance) // containerIDs are the keys to this map of instances

		containerIDAddressPairs := getAllContainerIdAddressPairs(latestHostServerResponse)

		// map of service instances by IP address
		serviceInstances := updateAmalgam8ServiceInstances()

		time.Sleep(1 * time.Second)

		// Add the pairs with Amalgam8 IP addresses to our collection
		for _, pa := range containerIDAddressPairs {
			// Check to see if this container is in the collection of Amalgam8 services
			if _, ok := serviceInstances[pa.IP]; ok { 
				var tmp = serviceInstances[pa.IP]
				tmp.ContainerID = pa.ID
				tmp.IPaddress = pa.IP
				serviceInstances[pa.IP] = tmp
				m[pa.ID] = tmp
				//log.Println(serviceInstances[pa.IP])
			}
		}
		serviceInstancesByContainerID = m
		log.Println("Added Container IDs to Amalgam8 services")
	}
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

// GetServiceInstances returns 
func GetServiceInstances(serviceName string) []serviceInstance {
	var svcDetails serviceDetails
	cmdArgs := []string{"-H 'Accept: application/json'","http://localhost:31300/api/v1/services/" + serviceName}
	nu, errrr := exec.Command("curl", cmdArgs...).Output()
	if errrr != nil {
		log.Fatal(errrr)
	}
	a := string(nu)
	json.Unmarshal([]byte(a), &svcDetails)
	return svcDetails.Instances
	/*var addressList []Address

	for _, instance := range svcDetails.Instances {
		ipport := strings.Split(instance.Endpoint.Value, ":")
		if len(ipport) != 2 {
			log.Println("issue with splitting address")
		}
		addressList = append(addressList, Address{IP:ipport[0],Port:ipport[1]})
	}
	return addressList*/
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
}