package main

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
    "time"

    "k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
)

func GetContainersFromKubernetesGoClient(clientset *kubernetes.Clientset) ([]ContainerSimple, error) {
    var containers []ContainerSimple
    podsResponse, err := clientset.Core().Pods("").List(api.ListOptions{})
    if err != nil {
        panic(err.Error())
    }
    for _, pod := range podsResponse.Items {
        for _, containerStatus := range pod.Status.ContainerStatuses {
            containerIDparts := strings.Split(containerStatus.ContainerID, "://")
            if len(containerIDparts) == 2 {
                containerID := containerIDparts[1]
                newContainer := ContainerSimple{IP: pod.Status.PodIP, ID: containerID, Name: containerStatus.Name}
                containers = append(containers, newContainer)
            }
        }
    }
    return containers, nil
}

func GetContainersFromKubernetes() ([]ContainerSimple, error) {
    pods, err := getPods()
    if err != nil {
        return []ContainerSimple{}, fmt.Errorf("getPods: %v", err)
    }
    containers := getContainersFromPods(pods...)
    return containers, nil
}

func getPods() ([]Pod, error) {
    var pods []Pod
    cmdArgs := []string{ClusterIP + "/api/v1/pods"}
	output, err := exec.Command("curl", cmdArgs...).Output()
	if err != nil {
		return pods, fmt.Errorf("getPods: ", err)
	}
	//s := string(output)
    //log.Println(s)
	var podDataResponse PodDataResponse
	json.Unmarshal([]byte(string(output)), &podDataResponse)

    for _, podData := range podDataResponse.Items {
        pods = append(pods, podData.toPod())
    }
    return pods, nil
}

func getContainersFromPods(pods ...Pod) []ContainerSimple {
    containers := []ContainerSimple{}
    for _, pod := range pods {
        containers = append(containers, pod.Containers...)
    }
    return containers
}

type Pod struct {
    Name string
    HostIP string
    IP string
    Containers []ContainerSimple
}

func (p *PodData) toPod() Pod {
    var containers []ContainerSimple
    for _, containerStatus := range p.Status.ContainerStatuses {
        containerIDparts := strings.Split(containerStatus.ContainerID, "://")
        if len(containerIDparts) == 2 {
            containerID := containerIDparts[1]
            newContainer := ContainerSimple{IP: p.Status.PodIP, ID: containerID}
            containers = append(containers, newContainer)
        }
    }
    return Pod{
        Name: p.Metadata.Name,
        HostIP: p.Status.HostIP,
        IP: p.Status.PodIP,
        Containers: containers,
    }
}

type PodData struct {
	Metadata struct {
		Name string `json:"name"`
		GenerateName string `json:"generateName"`
		Namespace string `json:"namespace"`
		SelfLink string `json:"selfLink"`
		UID string `json:"uid"`
		ResourceVersion string `json:"resourceVersion"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Labels struct {
			Name string `json:"name"`
		} `json:"labels"`
		Annotations struct {
			KubernetesIoCreatedBy string `json:"kubernetes.io/created-by"`
		} `json:"annotations"`
		OwnerReferences []struct {
			APIVersion string `json:"apiVersion"`
			Kind string `json:"kind"`
			Name string `json:"name"`
			UID string `json:"uid"`
			Controller bool `json:"controller"`
		} `json:"ownerReferences"`
	} `json:"metadata"`
	Spec struct {
		Volumes []struct {
			Name string `json:"name"`
			Secret struct {
				SecretName string `json:"secretName"`
				DefaultMode int `json:"defaultMode"`
			} `json:"secret"`
		} `json:"volumes"`
		Containers []struct {
			Name string `json:"name"`
			Image string `json:"image"`
			Ports []struct {
				ContainerPort int `json:"containerPort"`
				Protocol string `json:"protocol"`
			} `json:"ports,omitempty"`
			Env []struct {
				Name string `json:"name"`
				Value string `json:"value"`
			} `json:"env"`
			Resources struct {
			} `json:"resources"`
			VolumeMounts []struct {
				Name string `json:"name"`
				ReadOnly bool `json:"readOnly"`
				MountPath string `json:"mountPath"`
			} `json:"volumeMounts"`
			TerminationMessagePath string `json:"terminationMessagePath"`
			ImagePullPolicy string `json:"imagePullPolicy"`
		} `json:"containers"`
		RestartPolicy string `json:"restartPolicy"`
		TerminationGracePeriodSeconds int `json:"terminationGracePeriodSeconds"`
		DNSPolicy string `json:"dnsPolicy"`
		ServiceAccountName string `json:"serviceAccountName"`
		ServiceAccount string `json:"serviceAccount"`
		NodeName string `json:"nodeName"`
		SecurityContext struct {
		} `json:"securityContext"`
	} `json:"spec"`
	Status struct {
		Phase string `json:"phase"`
		Conditions []struct {
			Type string `json:"type"`
			Status string `json:"status"`
			LastProbeTime interface{} `json:"lastProbeTime"`
			LastTransitionTime time.Time `json:"lastTransitionTime"`
		} `json:"conditions"`
		HostIP string `json:"hostIP"`
		PodIP string `json:"podIP"`
		StartTime time.Time `json:"startTime"`
		ContainerStatuses []struct {
			Name string `json:"name"`
			State struct {
				Running struct {
					StartedAt time.Time `json:"startedAt"`
				} `json:"running"`
			} `json:"state"`
			LastState struct {
			} `json:"lastState"`
			Ready bool `json:"ready"`
			RestartCount int `json:"restartCount"`
			Image string `json:"image"`
			ImageID string `json:"imageID"`
			ContainerID string `json:"containerID"`
		} `json:"containerStatuses"`
	} `json:"status"`
}

type PodDataResponse struct {
	Kind string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Metadata struct {
		SelfLink string `json:"selfLink"`
		ResourceVersion string `json:"resourceVersion"`
	} `json:"metadata"`
	Items []PodData `json:"items"`
}