// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package genie provides API for single-networking or multi-networking.
It has genie-cadvisor-client that exposes an API to talk to cAdvisor.
It has genie-controller that exposes an API for pod single IP based
networking or pod multi-IP based networking.
*/
package genie

import (
	"encoding/json"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"os"
	//"os/exec"
	//"strings"
	"github.com/Huawei-PaaS/CNI-Genie/utils"
	"strconv"
	"strings"
)

type LogicalNetwork struct {
	apiVersion string `json:"apiVersion"`
	Metadata   struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		PhysicalNet string `json:"physicalNet,omitempty"`
		SubSubnet   string `json:"sub_subnet,omitempty"`
		Plugin      string `json:"plugin,omitempty"`
	} `json:"spec"`
}

type PhysicalNetwork struct {
	apiVersion string `json:"apiVersion"`
	Metadata   struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace,omitempty"`
	} `json:"metadata"`
	Spec struct {
		ReferNic     string `json:"refer_nic"`
		SharedStatus struct {
			Plugin          string `json:"plugin,omitempty"`
			Subnet          string `json:"subnet,omitempty"`
			DedicatedStatus bool   `json:"dedicatedNet"`
		} `json:"sharedStatus"`
	} `json:"spec"`
}

/**
Returns the list of plugins intended by user through physical network crd
	- annot : pod annotation received
*/
func GetPluginInfoFromPhysicalNw(phyNwName string, namespace string, client *kubernetes.Clientset, pluginInfo utils.PluginInfo) (utils.PluginInfo, error) {
	physicalNwPath := fmt.Sprintf("/apis/alpha.network.k8s.io/v1/namespaces/%s/physicalnetworks/%s", namespace, phyNwName)

	//fmt.Fprintf(os.Stderr, "CNI Genie networks out =%v, err=%v\n", out, err)
	fmt.Fprintf(os.Stderr, "CNI Genie physical newtwork self link=%v\n", physicalNwPath)
	physicalNwObj, err := client.ExtensionsV1beta1().RESTClient().Get().AbsPath(physicalNwPath).DoRaw()

	if err != nil {
		return pluginInfo, fmt.Errorf("CNI Genie failed to get physical network object for the network %v, namespace %v\n", phyNwName, namespace)
	}

	physicalNwInfo := PhysicalNetwork{}
	if err = json.Unmarshal(physicalNwObj, &physicalNwInfo); err != nil {
		return pluginInfo, fmt.Errorf("CNI Genie failed to physical network info: %v", err)
	}
	pluginInfo.Refer_nic = physicalNwInfo.Spec.ReferNic
	fmt.Fprintf(os.Stderr, "CNI Genie physicalNwInfo=%v\n", physicalNwInfo)
	if physicalNwInfo.Spec.SharedStatus.DedicatedStatus == true {

		pluginInfo.PluginName = physicalNwInfo.Spec.SharedStatus.Plugin
	}

	pluginInfo.Subnet = physicalNwInfo.Spec.SharedStatus.Subnet
	fmt.Fprintf(os.Stderr, "CNI Genie pluginInfo =%v\n", pluginInfo)
	return pluginInfo, nil
}

/**
Returns the list of plugins intended by user through network crd
	- annot : pod annotation received
*/
func GetPluginInfoFromNwAnnot(networkAnnot string, namespace string, client *kubernetes.Clientset) ([]utils.PluginInfo, error) {
	var pluginInfoList []utils.PluginInfo
	var networkName string

	logicalNwList := strings.Split(networkAnnot, ",")
	usedIntfMap := make(map[string]bool)
	i := 0
	pluginInfo := utils.PluginInfo{}

	for _, logicalNw := range logicalNwList {
		if true == strings.Contains(logicalNw, ":") {
			netNIfName := strings.Split(logicalNw, ":")
			networkName = netNIfName[0]
			pluginInfo.IfName = netNIfName[1]
		} else {
			networkName = logicalNw
			for {
				nic := "eth" + strconv.Itoa(i)
				if usedIntfMap[nic] == false {
					pluginInfo.IfName = nic
					usedIntfMap[nic] = true
					break
				}
				i++
			}
		}

		logicalNwPath := fmt.Sprintf("/apis/alpha.network.k8s.io/v1/namespaces/%s/logicalnetworks/%s", namespace,
			networkName)
		//fmt.Fprintf(os.Stderr, "CNI Genie networks out =%v, err=%v\n", out, err)
		fmt.Fprintf(os.Stderr, "CNI Genie logical newtwork self link=%v\n", logicalNwPath)
		logicalNwObj, err := client.ExtensionsV1beta1().RESTClient().Get().AbsPath(logicalNwPath).DoRaw()

		if err != nil {
			return pluginInfoList, fmt.Errorf("CNI Genie failed to get logical network object for the network %v, namespace %v\n", networkAnnot, namespace)
		}

		logicalNwInfo := LogicalNetwork{}
		if err = json.Unmarshal(logicalNwObj, &logicalNwInfo); err != nil {
			return pluginInfoList, fmt.Errorf("CNI Genie failed to logical network info: %v", err)
		}

		if logicalNwInfo.Spec.PhysicalNet == "" {
			return pluginInfoList, fmt.Errorf("CNI Genie failed to find physical network mapping in logical network %v, "+"namespace %v\n",
				networkName, namespace)
		}
		pluginInfo.PluginName = logicalNwInfo.Spec.Plugin

		pluginInfo, err := GetPluginInfoFromPhysicalNw(logicalNwInfo.Spec.PhysicalNet, namespace, client, pluginInfo)

		if logicalNwInfo.Spec.PhysicalNet == "" {
			pluginInfo.Subnet = logicalNwInfo.Spec.SubSubnet
		}
		fmt.Fprintf(os.Stderr, "CNI Genie pluginInfoList pluginInfo=%v\n", pluginInfo)

		pluginInfoList = append(pluginInfoList, pluginInfo)
	}
	fmt.Fprintf(os.Stderr, "CNI Genie pluginInfoList =%v\n", pluginInfoList)
	return pluginInfoList, nil
}
