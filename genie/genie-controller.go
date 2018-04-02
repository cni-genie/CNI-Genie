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
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Huawei-PaaS/CNI-Genie/plugins"
	"github.com/Huawei-PaaS/CNI-Genie/utils"
	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// MultiIPPreferencesAnnotation is a key used for parsing pod
	// definitions containing "multi-ip-preferences" annotation
	MultiIPPreferencesAnnotation = "multi-ip-preferences"
	DefaultNetDir                = "/etc/cni/net.d"
	// DefaultPluginDir specifies the default directory path for cni binary files
	DefaultPluginDir = "/opt/cni/bin"
	// ConfFilePermission specifies the default permission for conf file
	ConfFilePermission os.FileMode = 0644
)

// PopulateCNIArgs wraps skel.CmdArgs into Genie's native CNIArgs format.
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

// ParseCNIConf parses input configuration file and returns
// Genie's native NetConf object.
func ParseCNIConf(confData []byte) (utils.NetConf, error) {
	// Unmarshall the network config, and perform validation
	conf := utils.NetConf{}
	if err := json.Unmarshal(confData, &conf); err != nil {
		return conf, fmt.Errorf("failed to load netconf: %v", err)
	}
	return conf, nil
}

// AddPodNetwork adds pod networking. It has logic to parse each pod
// definition's annotations. It looks for container networking solutions (CNS)
// types passed as annotation in pod defintion. For every CNS types, it talks
// to corresponding CNS object and fetches an IP from it's IPAM.
// It also applies the IP as ethX inside the pod.
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

	// create kubeclient to talk to k8s api-server
	kubeClient, err := GetKubeClient(conf)
	if err != nil {
		return nil, fmt.Errorf("CNI Genie error at GetKubeClient: %v", err)
	}

	// parse pod annotations for cns types
	// eg:
	//    cni: "canal,weave"
	annots, err := ParsePodAnnotationsForCNI(kubeClient, k8sArgs, conf)
	if err != nil {
		return nil, fmt.Errorf("CNI Genie error at ParsePodAnnotations: %v", err)
	}

	// parse pod annotations for "multi-ip-preferences"
	// eg:
	//   multi-ip-preferences : |
	//       { "multi-entry": 0, "ips": {"":{"ip":"", "interfaces":""}}}
	multiIPPrefAnnot := ParsePodAnnotationsForMultiIPPrefs(kubeClient, k8sArgs)

	var newErr error
	var intfName string
	for i, pluginElement := range annots {
		if pluginElement.IfName != "" {
			intfName = pluginElement.IfName
		} else {
			intfName = "eth" + strconv.Itoa(i)
		}
		// fetches an IP from corresponding CNS IPAM and returns result object
		result, err = addNetwork(intfName, pluginElement.PluginName, cniArgs)
		fmt.Fprintf(os.Stderr, "CNI Genie addNetwork err *** %v result***  %v\n", err, result)
		if err != nil {
			newErr = err
		}
		// Update pod definition with IPs "multi-ip-preferences"
		multiIPPrefAnnot, err = UpdatePodDefinition(intfName, result, multiIPPrefAnnot, kubeClient, k8sArgs)
		if err != nil {
			newErr = err
		}
	}
	if newErr != nil {
		return nil, fmt.Errorf("CNI Genie error at addNetwork: %v", newErr)
	}
	return result, nil
}

// DeletePodNetwork deletes pod networking. It has logic to parse each pod
// definition's annotations. It looks for container networking solutions (CNS)
// types passed as annotation in pod defintion. For every CNS types, it talks
// to corresponding CNS object and releases an IP from it's IPAM.
func DeletePodNetwork(cniArgs utils.CNIArgs, conf utils.NetConf) error {
	k8sArgs, err := loadArgs(cniArgs)
	if err != nil {
		return fmt.Errorf("CNI Genie internal error at loadArgs: %v", err)
	}
	_, _, err = getIdentifiers(cniArgs, k8sArgs)
	if err != nil {
		return fmt.Errorf("CNI Genie internal error at getIdentifiers: %v", err)
	}

	// create kubeclient to talk to k8s api-server
	kubeClient, err := GetKubeClient(conf)
	if err != nil {
		return fmt.Errorf("CNI Genie error at GetKubeClient: %v", err)
	}

	// parse pod annotations for cns types
	// eg:
	//    cni: "canal,weave"
	annots, err := ParsePodAnnotationsForCNI(kubeClient, k8sArgs, conf)
	if err != nil {
		return fmt.Errorf("CNI Genie error at ParsePodAnnotations: %v", err)
	}

	var newErr error
	var intfName string
	for i, pluginElement := range annots {
		if pluginElement.IfName != "" {
			intfName = pluginElement.IfName
		} else {
			intfName = "eth" + strconv.Itoa(i)
		}
		// releases an IP from corresponding CNS IPAM and returns error if any exception
		err = deleteNetwork(intfName, pluginElement.PluginName, cniArgs)
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

// UpdatePodDefinition updates the pod definition with multi ip addresses.
// It updates pod definition with annotation containing multi ips from
// different configured networking solutions. It is also used in "nocni"
// case where ideal network has been chosen for the pod. Pod annotation
// in this case will update with CNS that's chosen at run time.
func UpdatePodDefinition(intfName string, result types.Result, multiIPPrefAnnot string, client *kubernetes.Clientset, k8sArgs utils.K8sArgs) (string, error) {
	var multiIPPreferences utils.MultiIPPreferences

	if multiIPPrefAnnot == "" {
		fmt.Fprintf(os.Stderr, "CNI Genie No multi-ip-preferences annotation\n")
		return multiIPPrefAnnot, nil
	}

	if err := json.Unmarshal([]byte(multiIPPrefAnnot), &multiIPPreferences); err != nil {
		fmt.Errorf("CNI Genie Error parsing MultiIPPreferencesAnnotation = %s\n", err)
	}

	currResult, err := current.NewResultFromResult(result)
	if err != nil {
		return multiIPPrefAnnot, fmt.Errorf("CNI Genie Error when converting result to current version = %s", err)
	}

	intfId, _ := strconv.Atoi(strings.Split(intfName, "eth")[1])
	multiIPPreferences.MultiEntry = multiIPPreferences.MultiEntry + 1
	multiIPPreferences.Ips["ip"+strconv.Itoa(intfId+1)] =
		utils.IPAddressPreferences{currResult.IPs[0].Address.IP.String(), intfName}

	tmpMultiIPPreferences, err := json.Marshal(&multiIPPreferences)

	if err != nil {
		return multiIPPrefAnnot, err
	}

	// Get pod defition to update it in next steps
	pod, err := GetPodDefinition(client, string(k8sArgs.K8S_POD_NAMESPACE), string(k8sArgs.K8S_POD_NAME))
	if err != nil {
		return multiIPPrefAnnot, err
	}

	multiIPPref := fmt.Sprintf(
		`{"metadata":{"annotations":{"%s":%s}}}`, MultiIPPreferencesAnnotation, strconv.Quote(string(tmpMultiIPPreferences)))

	fmt.Fprintf(os.Stderr, "CNI Genie pod.Annotations[MultiIPPreferencesAnnotation] after = %s\n", multiIPPref)
	pod, err = client.Pods(string(k8sArgs.K8S_POD_NAMESPACE)).Patch(pod.Name, api.StrategicMergePatchType, []byte(multiIPPref))
	if err != nil {
		return multiIPPrefAnnot, fmt.Errorf("CNI Genie Error updating pod = %s", err)
	}
	return string(tmpMultiIPPreferences), nil
}

// GetPodDefinition gets pod definition through k8s api server
func GetPodDefinition(client *kubernetes.Clientset, podNamespace string, podName string) (*v1.Pod, error) {
	pod, err := client.Pods(podNamespace).Get(fmt.Sprintf("%s", podName), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pod, nil
}

// GetKubeClient creates a kubeclient from genie-kubeconfig file,
// default location is /etc/cni/net.d.
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

//ParsePodAnnotationsForCNI does following tasks
//  - get pod definition
//  - parses annotation section for "cni"
//  - Returns string array of networking solutions
func ParsePodAnnotationsForCNI(client *kubernetes.Clientset, k8sArgs utils.K8sArgs, conf utils.NetConf) ([]utils.PluginInfo, error) {
	var annots []utils.PluginInfo

	annot, err := getK8sPodAnnotations(client, k8sArgs)
	if err != nil {
		return annots, err
	}
	fmt.Fprintf(os.Stderr, "CNI Genie annot= [%s]\n", annot)

	annots, err = parseCNIAnnotations(annot, client, k8sArgs, conf)

	return annots, err

}

// ParsePodAnnotationsForMultiIPPrefs does following tasks
// - get pod definition
// - parses annotation section for "multi-ip-preferences"
// - Returns string
func ParsePodAnnotationsForMultiIPPrefs(client *kubernetes.Clientset, k8sArgs utils.K8sArgs) string {
	annot, _ := getK8sPodAnnotations(client, k8sArgs)
	fmt.Fprintf(os.Stderr, "CNI Genie annot= [%s]\n", annot)
	multiIpAnno := annot[MultiIPPreferencesAnnotation]
	return multiIpAnno
}

// ParsePodAnnotationsForNetworks does following tasks
// - get pod definition
// - parses annotation section for "networks"
// - Returns string
func ParsePodAnnotationsForNetworks(client *kubernetes.Clientset, k8sArgs utils.K8sArgs) string {
	annot, _ := getK8sPodAnnotations(client, k8sArgs)
	fmt.Fprintf(os.Stderr, "CNI Genie annot= [%s]\n", annot)
	networks := annot["networks"]
	return networks
}

//  parseCNIAnnotations parses pod yaml defintion for "cni" annotations.
func parseCNIAnnotations(annot map[string]string, client *kubernetes.Clientset, k8sArgs utils.K8sArgs, conf utils.NetConf) ([]utils.PluginInfo, error) {
	var finalPluginInfos []utils.PluginInfo
	var pluginInfo utils.PluginInfo

	if len(annot) == 0 {
		plugin := defaultPlugin(conf)
		fmt.Fprintf(os.Stderr, "CNI Genie no annotations is given! Using default plugin: %s,  annot is %v\n", plugin, annot)
		finalPluginInfos = []utils.PluginInfo{
			{PluginName: plugin},
		}
	} else if strings.TrimSpace(annot["cni"]) != "" {
		cniAnnots := strings.Split(annot["cni"], ",")
		for _, pluginName := range cniAnnots {
			pluginInfo.PluginName = pluginName
			finalPluginInfos = append(finalPluginInfos, pluginInfo)
			pluginInfo = utils.PluginInfo{}
		}

		fmt.Fprintf(os.Stderr, "CNI Genie finalPluginInfos= %v\n", finalPluginInfos)
	} else if networksAnnot := ParsePodAnnotationsForNetworks(client, k8sArgs); networksAnnot != "" {
		fmt.Fprintf(os.Stderr, "CNI Genie networks annotation passed\n")

		var err error

		finalPluginInfos, err = GetPluginInfoFromNwAnnot(strings.TrimSpace(annot["networks"]), string(k8sArgs.K8S_POD_NAMESPACE), client)
		if err != nil {
			return finalPluginInfos, fmt.Errorf("CNI Genie GetPluginInfoFromNwAnnot err= %v\n", err)
		}
	} else {
		glog.V(6).Info("Inside no cni annotation, calling cAdvisor client to retrieve ideal network solution")
		//TODO (Kaveh): Get this cAdvisor URL from genie conf file
		cns, err := GetCNSOrderByNetworkBandwith("http://127.0.0.1:4194")
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie GetCNSOrderByNetworkBandwith err= %v\n", err)
			return finalPluginInfos, fmt.Errorf("CNI Genie failed to retrieve CNS list from cAdvisor = %v", err)
		}
		fmt.Fprintf(os.Stderr, "CNI Genie cns= %v\n", cns)
		pod, err := client.Pods(string(k8sArgs.K8S_POD_NAMESPACE)).Get(fmt.Sprintf("%s", k8sArgs.K8S_POD_NAME), metav1.GetOptions{})
		if err != nil {
			return finalPluginInfos, fmt.Errorf("CNI Genie Error updating pod = %s", err)
		}
		cni := fmt.Sprintf(`{"metadata":{"annotations":{"cni":"%s"}}}`, cns)
		pod, err = client.Pods(string(k8sArgs.K8S_POD_NAMESPACE)).Patch(pod.Name, api.StrategicMergePatchType, []byte(cni))
		if err != nil {
			return finalPluginInfos, fmt.Errorf("CNI Genie Error updating pod = %s", err)
		}
		podTmp, _ := client.Pods(string(k8sArgs.K8S_POD_NAMESPACE)).Get(fmt.Sprintf("%s", k8sArgs.K8S_POD_NAME), metav1.GetOptions{})
		fmt.Fprintf(os.Stderr, "CNI Genie pod.Annotations[cni] after = %s\n", podTmp.Annotations["cni"])
		//finalAnnots = []string{cns}
		finalPluginInfos = []utils.PluginInfo{
			{PluginName: cns},
		}
	}

	fmt.Fprintf(os.Stderr, "CNI Genie return finalPluginInfos = %v\n", finalPluginInfos)
	return finalPluginInfos, nil
}

func ParseCNIConfFromFile(filename string) (*libcni.NetworkConfigList, error) {
	var err error
	var confList *libcni.NetworkConfigList

	if strings.HasSuffix(filename, ".conflist") {
		confList, err = libcni.ConfListFromFile(filename)
		if err != nil {
			return nil, fmt.Errorf("Error loading CNI config list file %s: %v", filename, err)
		}
	} else {
		conf, err := libcni.ConfFromFile(filename)
		if err != nil {
			return nil, fmt.Errorf("Error loading CNI config file %s: %v", filename, err)
		}
		// Ensure the config has a "type" so we know what plugin to run.
		// Also catches the case where somebody put a conflist into a conf file.
		if conf.Network.Type == "" {
			return nil, fmt.Errorf("Error loading CNI config file %s: no 'type'; perhaps this is a .conflist?", filename)
		}

		confList, err = libcni.ConfListFromConf(conf)
		if err != nil {
			return nil, fmt.Errorf("Error converting CNI config file %s to list: %v", filename, err)
		}
	}
	if len(confList.Plugins) == 0 {
		return nil, fmt.Errorf("CNI config list %s has no networks", filename)
	}
	return confList, nil
}

// checkPluginBinary checks for existence of plugin binary file
func checkPluginBinary(cniName string) error {
	binaries, err := ioutil.ReadDir(DefaultPluginDir)
	if err != nil {
		return fmt.Errorf("CNI Genie Error while checking binary file for plugin %s: %v", cniName, err)
	}

	for _, bin := range binaries {
		if true == strings.Contains(bin.Name(), cniName) {
			return nil
		}
	}
	return fmt.Errorf("CNI Genie Error user requested for unsupported plugin type %s. Only supported are (Romana, weave, canal, calico, flannel, bridge, macvlan, sriov)", cniName)
}

// placeConfFile creates a conf file in the specified directory path
func placeConfFile(obj interface{}, cniName string) (string, []byte, error) {
	dataBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("CNI Genie Error while marshalling configuration object for plugin %s: %v", cniName, err)
	}

	confFile := fmt.Sprintf(DefaultNetDir+"/"+"10-%s"+".conf", cniName)
	err = ioutil.WriteFile(confFile, dataBytes, ConfFilePermission)
	if err != nil {
		return "", nil, fmt.Errorf("CNI Genie Error while writing default conf file for plugin %s: %v", cniName, err)
	}
	return confFile, dataBytes, nil
}

// createConfIfBinaryExists checks for the binary file for a cni type and creates the conf if binary exists
func createConfIfBinaryExists(cniName string) (*libcni.NetworkConfigList, error) {
	// Check for the corresponding binary file.
	// If binary is not present, then do not create the conf file
	if err := checkPluginBinary(cniName); err != nil {
		return nil, err
	}

	var pluginObj interface{}
	switch cniName {
	case plugins.BridgeNet:
		pluginObj = plugins.GetBridgeConfig()
		break
	case plugins.Macvlan:
		pluginObj = plugins.GetMacvlanConfig()
		break
	case plugins.SriovNet:
		pluginObj = plugins.GetSriovConfig()
		break
	default:
		return nil, fmt.Errorf("CNI Genie Error user requested for unsupported plugin type %s. Only supported are (Romana, weave, canal, calico, flannel, bridge, macvlan, sriov)", cniName)
	}

	confFile, confBytes, err := placeConfFile(&pluginObj, cniName)
	if err != nil {
		return nil, err
	}
	netConf, err := libcni.ConfFromBytes(confBytes)
	if err != nil {
		return nil, err
	}
	confList, err := libcni.ConfListFromConf(netConf)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "CNI Genie Placed default conf file (%s) for cni type %s.\n", confFile, cniName)

	return confList, nil
}

// addNetwork is a core function that delegates call to pull IP from a Container Networking Solution (CNI Plugin)
func addNetwork(intfName string, cniName string, cniArgs utils.CNIArgs) (types.Result, error) {
	var result types.Result
	var err error

	if os.Setenv("CNI_IFNAME", intfName) != nil {
		fmt.Fprintf(os.Stderr, "CNI_IFNAME Error\n")
	}

	fmt.Fprintf(os.Stderr, "CNI Genie cniName=%v intfName =%v\n", cniName, intfName)

	cniConfig := libcni.CNIConfig{Path: []string{DefaultPluginDir}}

	files, err := libcni.ConfFiles(DefaultNetDir, []string{".conf", ".conflist"})
	fmt.Fprintf(os.Stderr, "CNI Genie files =%v\n", files)
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	var cniType string
	var netConfigList *libcni.NetworkConfigList
	for _, confFile := range files {
		if strings.Contains(confFile, cniName) && cniName != "" {
			// Get the configuration info from the file. If the file does not
			// contain valid conf, then skip it and check for another
			confFromFile, err := ParseCNIConfFromFile(confFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "CNI Genie Error loading CNI config file %s= %v\n", confFile, err)
				continue
			}
			cniType = confFromFile.Plugins[0].Network.Type
			fmt.Fprintf(os.Stderr, "CNI Genie cniName file found!!!!!! confFromFile.Type =%s\n", cniType)
			netConfigList = confFromFile
			break
		}
	}

	// If corresponding conf file is not present, then check for the
	// corresponding binary and create a default conf file if binary is present
	if netConfigList == nil {
		netConfigList, err = createConfIfBinaryExists(cniName)
		if err != nil {
			return nil, err
		}
		cniType = cniName
	}

	fmt.Fprintf(os.Stderr, "CNI Genie cni type= %s\n", cniType)
	rtConf, err := runtimeConf(cniArgs, intfName)
	if err != nil {
		return nil, fmt.Errorf("CNI Genie couldn't convert cniArgs to RuntimeConf: %v\n", err)
	}
	fmt.Fprintf(os.Stderr, "CNI Genie runtime configuration = %+v\n", rtConf)

	result, err = cniConfig.AddNetworkList(netConfigList, rtConf)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "CNI Genie final result = %v\n", result)

	return result, nil
}

// deleteNetwork is a core function that delegates call to release IP from a Container Networking Solution (CNI Plugin)
func deleteNetwork(intfName string, cniName string, cniArgs utils.CNIArgs) error {
	var conf *libcni.NetworkConfigList

	if os.Setenv("CNI_IFNAME", intfName) != nil {
		fmt.Fprintf(os.Stderr, "CNI_IFNAME Error\n")
	}

	cniConfig := libcni.CNIConfig{Path: []string{DefaultPluginDir}}

	files, err := libcni.ConfFiles(DefaultNetDir, []string{".conf"})
	fmt.Fprintf(os.Stderr, "CNI Genie files =%v\n", files)
	switch {
	case err != nil:
		return err
	case len(files) == 0:
		return fmt.Errorf("No networks found in %s", DefaultNetDir)
	}
	sort.Strings(files)
	for _, confFile := range files {
		if strings.Contains(confFile, cniName) && cniName != "" {
			confFromFile, err := ParseCNIConfFromFile(confFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "CNI Genie Error loading CNI config file =%v\n", confFile, err)
				continue
			}
			fmt.Fprintf(os.Stderr, "CNI Genie cniName file found!!!!!! confFromFile.Type =%v\n", confFromFile.Plugins[0].Network.Type)

			conf = confFromFile
			fmt.Fprintf(os.Stderr, "CNI Genie cni type= %s\n", conf.Plugins[0].Network.Type)
			rtConf, err := runtimeConf(cniArgs, intfName)
			if err != nil {
				return fmt.Errorf("CNI Genie couldn't convert cniArgs to RuntimeConf: %v\n", err)
			}
			err = cniConfig.DelNetworkList(conf, rtConf)
			if err != nil {
				return err
			}
			break
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

func runtimeConf(cniArgs utils.CNIArgs, iface string) (*libcni.RuntimeConf, error) {
	k8sArgs, err := loadArgs(cniArgs)
	if err != nil {
		return nil, err
	}
	var ignoreUnknown string
	if k8sArgs.IgnoreUnknown {
		ignoreUnknown = "1"
	} else {
		ignoreUnknown = "0"
	}

	return &libcni.RuntimeConf{
		ContainerID: cniArgs.ContainerID,
		NetNS:       cniArgs.Netns,
		IfName:      iface,
		Args: [][2]string{
			{"IgnoreUnknown", ignoreUnknown},
			{"K8S_POD_NAMESPACE", string(k8sArgs.K8S_POD_NAMESPACE)},
			{"K8S_POD_NAME", string(k8sArgs.K8S_POD_NAME)},
			{"K8S_POD_INFRA_CONTAINER_ID", string(k8sArgs.K8S_POD_INFRA_CONTAINER_ID)},
		}}, nil
}

func defaultPlugin(conf utils.NetConf) string {
	if conf.DefaultPlugin == "" {
		return "weave"
	}
	return conf.DefaultPlugin
}
