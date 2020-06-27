/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"flag"
	"fmt"
	genieUtils "github.com/cni-genie/CNI-Genie/utils"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"os"
)

/* Info carrying plugin name and subnet ranges already used logical networks*/
type PluginSubnetUsageData struct {
	PluginName      string
	SubnetUsageData map[string]string
}

const (
	ERR_NO_PLUGIN_MENTIONED_IN_LOGICAL_NETWORK  = "No plugin mentioned in logical network"
	ERR_NO_PLUGIN_MENTIONED_IN_PHYSICAL_NETWORK = "No plugin mentioned in physical network"
	ERR_PHYSICAL_NW_NOT_FOUND                   = "Physical network not found"
	ERR_INVALID_INNER_SUBNET                    = "Invalid inner subnet range"
	ERR_SUBNET_OVERLAP_WITH_OTHER               = "Inner subnet overlap with some other logical network subnet"
	ERR_SUBNET_NOT_SPECIFIED                    = "Subnet not specified for logical network while using shared physical network"
	ERR_INCORRECT_PHYSICALNETWORK_PARAS         = "Incorrect physical network parameters"
	ERR_INTERNAL_PROCESSING_FAILED              = "Internal processing failure"
)

var PluginSubnetUsageDataList []PluginSubnetUsageData

var (
	masterURL  string
	kubeconfig string
)

func initURLs() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func GetKubeClient() (*kubernetes.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	return kubeClient, nil
}

/* Function to validate whether the 2 input subnets overlap with eachother */
func checkSubnetOverlap(firstSubnetStr, secondSubnetStr string) bool {

	var isFirstPartofSecond bool
	var isSecondPartofFirst bool
	isFirstPartofSecond = true
	isSecondPartofFirst = true

	_, firstSubnet, _ := net.ParseCIDR(firstSubnetStr)
	_, secondSubnet, _ := net.ParseCIDR(secondSubnetStr)

	for i := range secondSubnet.IP {
		if secondSubnet.IP[i]&secondSubnet.Mask[i] != firstSubnet.IP[i]&firstSubnet.Mask[i]&secondSubnet.Mask[i] {
			isFirstPartofSecond = false
		}
	}

	if true == isFirstPartofSecond {
		return true
	}

	for i := range firstSubnet.IP {
		if firstSubnet.IP[i]&firstSubnet.Mask[i] != secondSubnet.IP[i]&secondSubnet.Mask[i]&firstSubnet.Mask[i] {
			isSecondPartofFirst = false
		}
	}

	if true == isSecondPartofFirst {
		return true
	}

	/* This means 2 subnets do not overlap*/
	return false
}

/* Function to validate whether the innerSubnetStr is part of outer subnet or not*/
func checkIfValidInnerSubnet(outerSubnetStr, innerSubnetStr string) bool {

	_, outerSubnet, _ := net.ParseCIDR(outerSubnetStr)
	_, innerSubnet, _ := net.ParseCIDR(innerSubnetStr)

	for i := range outerSubnet.IP {
		if outerSubnet.IP[i]&outerSubnet.Mask[i] != innerSubnet.IP[i]&innerSubnet.Mask[i]&outerSubnet.Mask[i] {
			return false
		}
	}

	return true
}

// Validate logical network input.
func validateNetworkParas(logicalNetwork *genieUtils.LogicalNetwork) *v1beta1.AdmissionResponse {
	admissionResponse := v1beta1.AdmissionResponse{}
	var selectedPluginName string
	var selectedSubnet string

	/* If physical network name is not mentioned in logical network, then default plugin will be selected, so will
	not validate further */
	phyNwName := logicalNetwork.Spec.PhysicalNet
	if "" == phyNwName {
		glog.Info(" CNI Genie physical network not mentioned in logical network %s, "+
			"default will be used", logicalNetwork.ObjectMeta.Name)
		admissionResponse.Allowed = true
		return &admissionResponse
	}

	admissionResponse.Allowed = false

	client, error := GetKubeClient()
	if error != nil {
		admissionResponse.Result = &metav1.Status{
			Reason: ERR_INTERNAL_PROCESSING_FAILED,
		}
		return &admissionResponse
	}

	physicalNwPath := fmt.Sprintf("/apis/alpha.network.k8s.io/v1/namespaces/%s/physicalnetworks/%s",
		logicalNetwork.ObjectMeta.Namespace, phyNwName)

	physicalNwObj, err := client.ExtensionsV1beta1().RESTClient().Get().AbsPath(physicalNwPath).DoRaw()

	if err != nil {
		admissionResponse.Result = &metav1.Status{
			Reason: ERR_PHYSICAL_NW_NOT_FOUND,
		}
		return &admissionResponse
	}

	physicalNwInfo := genieUtils.PhysicalNetwork{}
	if err = json.Unmarshal(physicalNwObj, &physicalNwInfo); err != nil {
		admissionResponse.Result = &metav1.Status{
			Reason: ERR_INCORRECT_PHYSICALNETWORK_PARAS,
		}
		return &admissionResponse
	}

	fmt.Fprintf(os.Stderr, "CNI Genie physicalNwInfo=%v\n", physicalNwInfo)
	if physicalNwInfo.Spec.SharedStatus.DedicatedStatus == true {

		selectedPluginName := physicalNwInfo.Spec.SharedStatus.Plugin

		if "" == selectedPluginName {
			admissionResponse.Result = &metav1.Status{
				Reason: ERR_NO_PLUGIN_MENTIONED_IN_PHYSICAL_NETWORK,
			}

			return &admissionResponse

		}

		outerSubnet := physicalNwInfo.Spec.SharedStatus.Subnet

		isvalidSubnet := checkIfValidInnerSubnet(outerSubnet, logicalNetwork.Spec.SubSubnet)
		if false == isvalidSubnet {

			admissionResponse.Result = &metav1.Status{
				Reason: ERR_INVALID_INNER_SUBNET,
			}
			return &admissionResponse
		}
		selectedSubnet = logicalNetwork.Spec.SubSubnet
	} else {
		/* Incase of shared physical network, subnet and plugin must be specified as part of logical network*/
		if "" == logicalNetwork.Spec.Plugin {
			admissionResponse.Result = &metav1.Status{
				Reason: ERR_NO_PLUGIN_MENTIONED_IN_LOGICAL_NETWORK,
			}
			return &admissionResponse

		}
		if "" == logicalNetwork.Spec.SubSubnet {
			admissionResponse.Result = &metav1.Status{
				Reason: ERR_SUBNET_NOT_SPECIFIED,
			}
			return &admissionResponse

		}

		selectedPluginName = logicalNetwork.Spec.Plugin
		selectedSubnet = logicalNetwork.Spec.SubSubnet
	}

	isPluginFound := false
	var pluginSubnetUsageData PluginSubnetUsageData

	/* Check whether the subnet is already part of any other logical network*/
	if nil != PluginSubnetUsageDataList {
		for i := range PluginSubnetUsageDataList {
			if PluginSubnetUsageDataList[i].PluginName == selectedPluginName {
				isPluginFound = true
				pluginSubnetUsageData = PluginSubnetUsageDataList[i]
				break
			}

		}
	}

	/* First time, entry for this plugin is getting added */
	if false == isPluginFound {

		pluginSubnetUsageData.PluginName = selectedPluginName
		pluginSubnetUsageData.SubnetUsageData = make(map[string]string)
		pluginSubnetUsageData.SubnetUsageData[selectedSubnet] = logicalNetwork.ObjectMeta.Name
		PluginSubnetUsageDataList = append(PluginSubnetUsageDataList, pluginSubnetUsageData)
	} else {
		/* Check whether selectedSubnet is already part of any of the existing subnets of this plugin*/
		if "" != pluginSubnetUsageData.SubnetUsageData[selectedSubnet] {
			admissionResponse.Result = &metav1.Status{
				Reason: ERR_SUBNET_OVERLAP_WITH_OTHER,
			}
			return &admissionResponse
		}
		for key := range pluginSubnetUsageData.SubnetUsageData {
			if checkSubnetOverlap(selectedSubnet, key) {
				admissionResponse.Result = &metav1.Status{
					Reason: ERR_SUBNET_OVERLAP_WITH_OTHER,
				}
				return &admissionResponse
			}
		}

		pluginSubnetUsageData.SubnetUsageData[selectedSubnet] = logicalNetwork.ObjectMeta.Name
	}

	admissionResponse.Allowed = true
	return &admissionResponse
}
