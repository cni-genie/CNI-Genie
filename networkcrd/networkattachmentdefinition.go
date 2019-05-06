package networkcrd

import (
	"encoding/json"
	"fmt"
	client "github.com/cni-genie/CNI-Genie/client"
	it "github.com/cni-genie/CNI-Genie/interfaces"
	"github.com/cni-genie/CNI-Genie/utils"
	"github.com/containernetworking/cni/libcni"
	"net"
	"os"
	"regexp"
	"strings"
)

func matchRegex(str string) error {
	exp := "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
	if str != "" {
		matched, _ := regexp.MatchString(exp, str)
		if matched == false {
			return fmt.Errorf("Expression mismatch: must consist of lower case alphanumeric characters and must start and end with an alphanumeric character")
		}
	}
	return nil
}

func validateIfName(ifName string) error {
	return matchRegex(ifName)
}

func validateFields(network NetworkSelectionElement) error {
	/* validation of network name and namespace are being purposefully skipped*/
	if len(network.IPs) > 0 {
		for _, ip := range network.IPs {
			if nil == net.ParseIP(ip) {
				return fmt.Errorf("Invalid IP address: %s", ip)
			}
		}
	}

	if network.Mac != "" {
		if _, err := net.ParseMAC(network.Mac); err != nil {
			return fmt.Errorf("Invalid mac address %s", network.Mac)
		}
	}

	if network.Interface != "" {
		if err := validateIfName(network.Interface); err != nil {
			return fmt.Errorf("Invalid interface name %s", network.Interface)
		}
	}

	return nil
}

func parseNetworkInfoFromAnnot(network *NetworkSelectionElement, annot string) {
	ns := strings.Index(annot, "/")
	if ns >= 0 {
		network.Namespace = annot[:ns]
	}
	network.Name = annot[ns+1:]
	ni := strings.LastIndex(network.Name, utils.IfNameDelimiter)
	if ni >= 0 {
		network.Interface = network.Name[ni+1:]
		network.Name = network.Name[:ni]
	}
}

func GetNetworkInfo(annotation, podNs string) ([]NetworkSelectionElement, error) {
	var networks []NetworkSelectionElement
	if true == strings.ContainsAny(annotation, "[{") {
		err := json.Unmarshal([]byte(annotation), &networks)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshalling network annotation: %v", err)
		}

		for i := range networks {
			if networks[i].Namespace == "" {
				networks[i].Namespace = podNs
			}
			if err = validateFields(networks[i]); err != nil {
				return nil, fmt.Errorf("Error in validation: %v", err)
			}
		}
	} else {
		nw := strings.Split(annotation, ",")
		for _, n := range nw {
			n = strings.TrimSpace(n)
			var network NetworkSelectionElement
			parseNetworkInfoFromAnnot(&network, n)
			if network.Namespace == "" {
				network.Namespace = podNs
			}
			if err := validateFields(network); err != nil {
				return nil, fmt.Errorf("Error in validation: %v", err)
			}

			networks = append(networks, network)
		}
	}

	return networks, nil
}

type optionalParameters struct {
	Cni struct {
		Ips []string `json:"ips,omitempty"`
		Mac string   `json:"mac,omitempty"`
	} `json:"cni"`
}

func GetConfigFromFile(networkCrd *NetworkAttachmentDefinition, cniDir string) (*libcni.NetworkConfigList, error) {
	return libcni.LoadConfList(cniDir, networkCrd.Name)
}

func GetConfigFromSpec(networkCrd *NetworkAttachmentDefinition, cni it.CNI) (*libcni.NetworkConfigList, error) {
	config := make(map[string]interface{})
	configbytes := []byte(networkCrd.Spec.Config)
	var netConfigList *libcni.NetworkConfigList
	err := json.Unmarshal(configbytes, &config)
	if err != nil {
		return nil, fmt.Errorf("Error parsing plugin configuration data: %v", err)
	}

	if name, ok := config["name"].(string); !ok || strings.TrimSpace(name) == "" {
		config["name"] = networkCrd.Name
		configbytes, err = json.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("Error inserting name into config: %v", err)
		}
	}

	if _, ok := config["plugins"]; ok {
		netConfigList, err = cni.ConfListFromBytes(configbytes)
		if err != nil {
			return nil, fmt.Errorf("Error getting conflist from bytes: %v", err)
		}
	} else {
		netConfigList, err = cni.ConfListFromConfBytes(configbytes)
		if err != nil {
			return nil, fmt.Errorf("Error converting conf bytes to conflist: %v", err)
		}
	}

	return netConfigList, nil
}

func GetNetworkCRDObject(kubeClient *client.KubeClient, name, namespace string) (*NetworkAttachmentDefinition, error) {
	path := fmt.Sprintf("/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s", namespace, name)
	fmt.Fprintf(os.Stderr, "CNI Genie network attachment definition object (%s:%s) path: %s\n", namespace, name, path)
	obj, err := kubeClient.GetRaw(path)
	if err != nil {
		return nil, fmt.Errorf("Error performing GET request: %v", err)
	}
	networkcrd := &NetworkAttachmentDefinition{}
	err = json.Unmarshal(obj, networkcrd)
	if err != nil {
		return nil, err
	}

	return networkcrd, nil
}
