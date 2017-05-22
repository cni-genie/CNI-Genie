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

package genie

import (
	"encoding/json"
	"fmt"
	"github.com/Huawei-PaaS/CNI-Genie/utils"
	"github.com/containernetworking/cni/pkg/ipam"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strconv"
	"strings"
)

const (
	// MultiIPPreferencesAnnotation is a key used for parsing pod
	// definitions containing "multi-ip-preferences" annotation
	MultiIPPreferencesAnnotation = "multi-ip-preferences"
)

func PopulateCNIArgs(args *skel.CmdArgs) utils.CNIArgs {
	cniArgs := utils.CNIArgs{}
	cniArgs.Args = args.Args
	cniArgs.StdinData = args.StdinData
	cniArgs.Path = args.Path
	cniArgs.Netns = args.Netns
	cniArgs.ContainerID = args.ContainerID
	cniArgs.IfName = args.IfName

	return cniArgs
}

func ParseCNIConf(confData []byte) (utils.NetConf, error) {
	// Unmarshall the network config, and perform validation
	conf := utils.NetConf{}
	if err := json.Unmarshal(confData, &conf); err != nil {
		return conf, fmt.Errorf("failed to load netconf: %v", err)
	}
	return conf, nil
}

/**
AddPodNetwork add pod networking by parsing container
networking solutions passed as pod's annotation
*/
func AddPodNetwork(cniArgs utils.CNIArgs, conf utils.NetConf) (types.Result, error) {
	// Collect the result in this variable - this is ultimately what gets "returned" by this function by printing
	// it to stdout.
	var result types.Result

	k8sArgs, err := loadArgs(cniArgs)
	if err != nil {
		return nil, fmt.Errorf("CNI Genie internal error at loadArgs: %v", err)
	}
	_, _, err = getIdentifiers(cniArgs, k8sArgs)
	if err != nil {
		return nil, fmt.Errorf("CNI Genie internal error at getIdentifiers: %v", err)
	}

	//Get KubeClient
	kubeClient, err := GetKubeClient(conf)
	if err != nil {
		return nil, fmt.Errorf("CNI Genie error at GetKubeClient: %v", err)
	}
	annots, err := ParsePodAnnotationsForCNI(kubeClient, k8sArgs)
	if err != nil {
		return nil, fmt.Errorf("CNI Genie error at ParsePodAnnotations: %v", err)
	}

	multiIPPrefAnnot := ParsePodAnnotationsForMultiIPPrefs(kubeClient, k8sArgs)

	var newErr error
	for i, ele := range annots {
		// in case of multi network or multi-ip-per-pod
		// we should always reinitalize the conf to
		// original value that came from StdinData
		conf, err = ParseCNIConf(cniArgs.StdinData)
		if err != nil {
			newErr = err
		}

		result, err = addNetwork(conf, i, ele, cniArgs)
		fmt.Fprintf(os.Stderr, "CNI Genie Error addNetwork err *** %v\n", err)
		fmt.Fprintf(os.Stderr, "CNI Genie Error addNetwork result***  %v\n", result)
		if err != nil {
			newErr = err
		}
		err = UpdatePodDefinition(i, result, multiIPPrefAnnot, kubeClient, k8sArgs)
		if err != nil {
			newErr = err
		}
	}
	if newErr != nil {
		return nil, fmt.Errorf("CNI Genie error at addNetwork: %v", newErr)
	}
	return result, nil
}

/**
DeletePodNetwork deletes pod networking by parsing container
networking solutions passed as pod's annotation
*/
func DeletePodNetwork(cniArgs utils.CNIArgs, conf utils.NetConf) error {
	k8sArgs, err := loadArgs(cniArgs)
	if err != nil {
		return fmt.Errorf("CNI Genie internal error at loadArgs: %v", err)
	}
	_, _, err = getIdentifiers(cniArgs, k8sArgs)
	if err != nil {
		return fmt.Errorf("CNI Genie internal error at getIdentifiers: %v", err)
	}

	kubeClient, err := GetKubeClient(conf)
	if err != nil {
		return fmt.Errorf("CNI Genie error at GetKubeClient: %v", err)
	}
	annots, err := ParsePodAnnotationsForCNI(kubeClient, k8sArgs)
	if err != nil {
		return fmt.Errorf("CNI Genie error at ParsePodAnnotations: %v", err)
	}

	var newErr error
	for i, ele := range annots {
		// in case of multi network or multi-ip-per-pod
		// we should always reinitalize the conf to
		// original value that came from StdinData
		conf, err = ParseCNIConf(cniArgs.StdinData)
		if err != nil {
			newErr = err
		}
		err = deleteNetwork(conf, i, ele, cniArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie Error deleteNetwork %v", err)
			newErr = err
		}
	}
	if newErr != nil {
		return fmt.Errorf("CNI Genie error at deleteNetwork: %v", newErr)
	}
	return nil
}

/**
UpdatePodDefinition updates the pod definition with multi ip addresses
It updates pod definition with annotation containing multi ips from
different configured networking solutions.
*/
func UpdatePodDefinition(intfId int, result types.Result, multiIPPrefAnnot string, client *kubernetes.Clientset, k8sArgs utils.K8sArgs) error {
	var multiIPPreferences utils.MultiIPPreferences

	if multiIPPrefAnnot == "" {
		fmt.Fprintf(os.Stderr, "CNI Genie No multi-ip-preferences annotation\n")
		return nil
	}

	if err := json.Unmarshal([]byte(multiIPPrefAnnot), &multiIPPreferences); err != nil {
		fmt.Errorf("CNI Genie Error parsing MultiIPPreferencesAnnotation = %s\n", err)
	}
	multiIPPreferences.MultiEntry = multiIPPreferences.MultiEntry + 1
	//TODO (Kaveh/Karun): Need some clean up here
	multiIPPreferences.Ips["ip"+strconv.Itoa(intfId+1)] =
		utils.IPAddressPreferences{
			strings.Split((strings.Split(result.String(), "IP4:{IP:{IP:")[1]),
				" Mask")[0], "eth" + strconv.Itoa(intfId)}

	tmpMultiIPPreferences, err := json.Marshal(&multiIPPreferences)

	if err != nil {
		return err
	}

	pod, err := GetPodDefinition(client, string(k8sArgs.K8S_POD_NAMESPACE), string(k8sArgs.K8S_POD_NAME))
	if err != nil {
		return err
	}

	pod.Annotations[MultiIPPreferencesAnnotation] = string(tmpMultiIPPreferences)
	fmt.Fprintf(os.Stderr, "CNI Genie pod.Annotations[MultiIPPreferencesAnnotation] after = %v\n", pod.Annotations[MultiIPPreferencesAnnotation])
	pod, err = client.Pods(string(k8sArgs.K8S_POD_NAMESPACE)).Update(pod)
	if err != nil {
		return fmt.Errorf("CNI Genie Error updating pod = %s", err)
	}
	return nil
}

/**
GetPodDefinition returns a pod definition
*/
func GetPodDefinition(client *kubernetes.Clientset, podNamespace string, podName string) (*v1.Pod, error) {
	pod, err := client.Pods(podNamespace).Get(fmt.Sprintf("%s", podName), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pod, nil
}

/**
GetKubeClient creates a kubeclient
*/
func GetKubeClient(conf utils.NetConf) (*kubernetes.Clientset, error) {
	// Some config can be passed in a kubeconfig file
	kubeconfig := conf.Kubernetes.Kubeconfig

	// Config can be overridden by config passed in explicitly in the network config.
	configOverrides := &clientcmd.ConfigOverrides{}

	// If an API root is given, make sure we're using using the name / port rather than
	// the full URL. Earlier versions of the config required the full `/api/v1/` extension,
	// so split that off to ensure compatibility.
	conf.Policy.K8sAPIRoot = strings.Split(conf.Policy.K8sAPIRoot, "/api/")[0]

	var overridesMap = []struct {
		variable *string
		value    string
	}{
		{&configOverrides.ClusterInfo.Server, conf.Policy.K8sAPIRoot},
		{&configOverrides.AuthInfo.ClientCertificate, conf.Policy.K8sClientCertificate},
		{&configOverrides.AuthInfo.ClientKey, conf.Policy.K8sClientKey},
		{&configOverrides.ClusterInfo.CertificateAuthority, conf.Policy.K8sCertificateAuthority},
		{&configOverrides.AuthInfo.Token, conf.Policy.K8sAuthToken},
	}

	// Using the override map above, populate any non-empty values.
	for _, override := range overridesMap {
		if override.value != "" {
			*override.variable = override.value
		}
	}

	// Also allow the K8sAPIRoot to appear under the "kubernetes" block in the network config.
	if conf.Kubernetes.K8sAPIRoot != "" {
		configOverrides.ClusterInfo.Server = conf.Kubernetes.K8sAPIRoot
	}

	// Use the kubernetes client code to load the kubeconfig file and combine it with the overrides.
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		configOverrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stderr, "Kubernetes config %v", config)

	// Create the clientset
	return kubernetes.NewForConfig(config)
}

/**
ParsePodAnnotationsForCNI does following tasks
 - get pod definition
 - parses annotation section for "cni"
 - Returns string array of networking solutions
*/
func ParsePodAnnotationsForCNI(client *kubernetes.Clientset, k8sArgs utils.K8sArgs) ([]string, error) {
	var annots []string

	annot, err := getK8sPodAnnotations(client, k8sArgs)
	if err != nil {
		return annots, err
	}
	fmt.Fprintf(os.Stderr, "CNI Genie annot= [%s]\n", annot)

	annots, err = parseCNIAnnotations(annot, client, k8sArgs)

	return annots, nil

}

/**
ParsePodAnnotationsForMultiIPPrefs does following tasks
 - get pod definition
 - parses annotation section for "multi-ip-preferences"
 - Returns string
*/
func ParsePodAnnotationsForMultiIPPrefs(client *kubernetes.Clientset, k8sArgs utils.K8sArgs) string {
	annot, _ := getK8sPodAnnotations(client, k8sArgs)
	fmt.Fprintf(os.Stderr, "CNI Genie annot= [%s]\n", annot)
	multiIpAnno := annot[MultiIPPreferencesAnnotation]
	return multiIpAnno
}

func parseCNIAnnotations(annot map[string]string, client *kubernetes.Clientset, k8sArgs utils.K8sArgs) ([]string, error) {
	var finalAnnots []string

	if len(annot) == 0 {
		fmt.Fprintf(os.Stderr, "CNI Genie no annotations is given! Default plugin is canal! annot is %V\n", annot)
		finalAnnots = []string{"canal"}
	} else if strings.TrimSpace(annot["cni"]) == "" {
		glog.V(6).Info("Inside no cni annotation, calling cAdvisor client to retrieve ideal network solution")
		//TODO (Kaveh): Get this cAdvisor URL from genie conf file
		cns, err := GetCNSOrderByNetworkBandwith("http://127.0.0.1:4194")
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie GetCNSOrderByNetworkBandwith err= %v\n", err)
			return finalAnnots, fmt.Errorf("CNI Genie failed to retrieve CNS list from cAdvisor = %v", err)
		}
		fmt.Fprintf(os.Stderr, "CNI Genie cns= %v\n", cns)
		pod, err := client.Pods(string(k8sArgs.K8S_POD_NAMESPACE)).Get(fmt.Sprintf("%s", k8sArgs.K8S_POD_NAME), metav1.GetOptions{})
		if err != nil {
			return finalAnnots, fmt.Errorf("CNI Genie Error updating pod = %s", err)
		}
		fmt.Fprintf(os.Stderr, "CNI Genie pod.Annotations[cni] before = %s\n", pod.Annotations["cni"])
		pod.Annotations["cni"] = cns
		pod, err = client.Pods(string(k8sArgs.K8S_POD_NAMESPACE)).Update(pod)
		if err != nil {
			return finalAnnots, fmt.Errorf("CNI Genie Error updating pod = %s", err)
		}
		podTmp, _ := client.Pods(string(k8sArgs.K8S_POD_NAMESPACE)).Get(fmt.Sprintf("%s", k8sArgs.K8S_POD_NAME), metav1.GetOptions{})
		fmt.Fprintf(os.Stderr, "CNI Genie pod.Annotations[cni] after = %s\n", podTmp.Annotations["cni"])
		finalAnnots = []string{cns}
	} else {
		finalAnnots = strings.Split(annot["cni"], ",")
		fmt.Fprintf(os.Stderr, "CNI Genie annots= %v\n", finalAnnots)
	}
	fmt.Fprintf(os.Stderr, "CNI Genie return finalAnnots = %v\n", finalAnnots)
	return finalAnnots, nil
}

func addNetwork(conf utils.NetConf, intfId int, cniName string, cniArgs utils.CNIArgs) (types.Result, error) {
	var result types.Result
	var stdinData []byte
	var err error
	if os.Setenv("CNI_IFNAME", "eth"+strconv.Itoa(intfId)) != nil {
		fmt.Fprintf(os.Stderr, "CNI_IFNAME Error\n")
	}
	fmt.Fprintf(os.Stderr, "CNI Genie cniName=%v\n", cniName)
	switch strings.TrimSpace(cniName) {
	case "romana":
		conf.Name = "romana-k8s-network" //romana expects this name!
		conf.IPAM.Type = "romana-ipam"
		conf.Type = "romana"
		stdinData, _ = json.Marshal(&conf)
		result, err = ipam.ExecAdd("romana", stdinData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie err = %v\n", err)
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "CNI Genie romana result = %v\n", result)
	case "weave":
		conf.Type = "weave-net"
		stdinData, _ = json.Marshal(&conf)
		result, err = ipam.ExecAdd("weave-net", stdinData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie err = %v\n", err)
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "CNI Genie weave result = %v\n", result)
	case "calico":
		conf.Type = "calico"
		conf.IPAM.Type = "host-local"
		conf.IPAM.Subnet = "usePodCidr"
		stdinData, _ = json.Marshal(&conf)
		result, err = ipam.ExecAdd("calico", stdinData)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "CNI Genie weave result = %v\n", result)
	case "canal":
		conf.Type = "calico"
		conf.IPAM.Type = "host-local"
		conf.IPAM.Subnet = "usePodCidr"
		stdinData, _ = json.Marshal(&conf)
		result, err = ipam.ExecAdd("calico", stdinData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie err = %v\n", err)
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "CNI Genie canal result = %v\n", result)
	case "flannel":
		conf.Type = "flannel"
		conf.Delegate.DelegateType = "flannel"
		conf.Delegate.EtcdEndpoints = conf.EtcdEndpoints
		conf.Delegate.LogLevel = conf.LogLevel
		conf.Delegate.Policy = conf.Policy
		conf.Delegate.Kubernetes = conf.Kubernetes
		stdinData, _ = json.Marshal(&conf)
		result, err = ipam.ExecAdd("flannel", stdinData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie err = %v\n", err)
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "CNI Genie flannel result = %v\n", result)
	}

	if result != nil {
		fmt.Fprintf(os.Stderr, "CNI Genie final result = %v\n", result)
		return result, nil
	}

	return nil, fmt.Errorf("CNI Genie doesn't support passed cniName [%v]. Only supported are (Romana, weave, canal, calico, flannel) \n", cniName)
}

func deleteNetwork(conf utils.NetConf, intfId int, cniName string, cniArgs utils.CNIArgs) error {
	var stdinData []byte

	if os.Setenv("CNI_IFNAME", "eth"+strconv.Itoa(intfId)) != nil {
		fmt.Fprintf(os.Stderr, "CNI_IFNAME Error\n")
	}
	switch strings.TrimSpace(cniName) {
	case "romana":
		conf.IPAM.Type = "romana-ipam"
		conf.Type = "romana"
		stdinData, _ = json.Marshal(&conf)
		ipamErr := ipam.ExecDel("romana", stdinData)
		if ipamErr != nil {
			return ipamErr
		}
	case "weave":
		conf.Type = "weave-net"
		stdinData, _ = json.Marshal(&conf)
		ipamErr := ipam.ExecDel("weave-net", stdinData)
		if ipamErr != nil {
			return ipamErr
		}
	case "calico":
		conf.Type = "calico"
		conf.IPAM.Type = "host-local"
		conf.IPAM.Subnet = "usePodCidr"
		stdinData, _ = json.Marshal(&conf)
		ipamErr := ipam.ExecDel("calico", stdinData)
		if ipamErr != nil {
			return ipamErr
		}
	case "canal":
		conf.Type = "calico"
		conf.IPAM.Type = "host-local"
		conf.IPAM.Subnet = "usePodCidr"
		stdinData, _ = json.Marshal(&conf)
		ipamErr := ipam.ExecDel("calico", stdinData)
		if ipamErr != nil {
			return ipamErr
		}
	case "flannel":
		conf.Type = "flannel"
		conf.Delegate.DelegateType = "flannel"
		conf.Delegate.EtcdEndpoints = conf.EtcdEndpoints
		conf.Delegate.LogLevel = conf.LogLevel
		conf.Delegate.Policy = conf.Policy
		conf.Delegate.Kubernetes = conf.Kubernetes
		stdinData, _ = json.Marshal(&conf)
		ipamErr := ipam.ExecDel("flannel", stdinData)
		if ipamErr != nil {
			return ipamErr
		}
	}
	return nil
}

func loadArgs(cniArgs utils.CNIArgs) (utils.K8sArgs, error) {
	k8sArgs := utils.K8sArgs{}
	err := types.LoadArgs(cniArgs.Args, &k8sArgs)
	if err != nil {
		return k8sArgs, err
	}
	return k8sArgs, nil
}

func getIdentifiers(cniArgs utils.CNIArgs, k8sArgs utils.K8sArgs) (workloadID string, orchestratorID string, err error) {
	// Determine if running under k8s by checking the CNI args
	if string(k8sArgs.K8S_POD_NAMESPACE) != "" && string(k8sArgs.K8S_POD_NAME) != "" {
		workloadID = fmt.Sprintf("%s.%s", k8sArgs.K8S_POD_NAMESPACE, k8sArgs.K8S_POD_NAME)
		orchestratorID = "k8s"
	} else {
		workloadID = cniArgs.ContainerID
		orchestratorID = "cni"
	}
	fmt.Fprintf(os.Stderr, "CNI Genie workloadID= %s\n", workloadID)
	fmt.Fprintf(os.Stderr, "CNI Genie orchestratorID= %s\n", orchestratorID)
	return workloadID, orchestratorID, nil
}

func getK8sPodAnnotations(client *kubernetes.Clientset, k8sArgs utils.K8sArgs) (map[string]string, error) {
	pod, err := GetPodDefinition(client, string(k8sArgs.K8S_POD_NAMESPACE), string(k8sArgs.K8S_POD_NAME))
	if err != nil {
		return nil, err
	}

	return pod.Annotations, nil
}
