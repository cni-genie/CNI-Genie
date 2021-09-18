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
	"net"
	"os"
	"strconv"
	"strings"

	"errors"
	"github.com/cni-genie/CNI-Genie/client"
	it "github.com/cni-genie/CNI-Genie/interfaces"
	"github.com/cni-genie/CNI-Genie/plugins"
	"github.com/cni-genie/CNI-Genie/utils"
	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sync"
)

const (
	// MultiIPPreferencesAnnotation is a key used for parsing pod
	// definitions containing "multi-ip-preferences" annotation
	MultiIPPreferencesAnnotation = "multi-ip-preferences"
	DefaultNetDir                = "/etc/cni/net.d"
	// DefaultPluginDir specifies the default directory path for cni binary files
	DefaultPluginDir                   = "/opt/cni/bin"
	MultiIPPreferencesAnnotationFormat = `{"multi_entry": 0,"ips": {}}`
	// SupportedPlugins lists the plugins supported by Genie
	SupportedPlugins = "bridge, calico, canal, flannel, macvlan, Romana, sriov, weave"
	// DefaultIfNamePrefix specifies the default prefix to be used while generating interface names
	DefaultIfNamePrefix = "eth"
	// NetworkAttachmentDefinitionAnnot specifies the pod Network Attachment Selection Annotation
	NetworkAttachmentDefinitionAnnot = "k8s.v1.cni.cncf.io/networks"
	// NetworkAttachmentStatusAnnot specifies the network attachment status annotation in pod objent
	NetworkAttachmentStatusAnnot = "k8s.v1.cni.cncf.io/network-status"
	err_nopod_novar              = "No pod or env var found"
	// Default value for cni version
	DefaultCNIVersion = "0.3.0"
)

type SetStatus func(current.Result, string, string, interface{}) interface{}

type sendCh struct {
	name   string
	ifName string
	res    types.Result
}

type GenieController struct {
	Invoke it.InvokeExec
	Cfg    *it.CNIConfig
	Kc     *client.KubeClient
	Cad    Cadvisor
}

// PopulateCNIArgs wraps skel.CmdArgs into Genie's native CNIArgs format.
func PopulateCNIArgs(args *skel.CmdArgs) *utils.CNIArgs {
	cniArgs := utils.CNIArgs{}
	cniArgs.Args = args.Args
	cniArgs.StdinData = args.StdinData
	cniArgs.Path = args.Path
	cniArgs.Netns = args.Netns
	cniArgs.ContainerID = args.ContainerID
	cniArgs.IfName = args.IfName

	return &cniArgs
}

func NewGenieController(conf *utils.GenieConf) (*GenieController, error) {
	kc, err := client.BuildKubeClientFromConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("Error building kubernetes client: %v", err)
	}
	return &GenieController{
		Kc: kc,
		Cfg: &it.CNIConfig{
			RW:     &it.IO{},
			CNI:    &it.Cni{},
			NetDir: DefaultNetDir,
			BinDir: DefaultPluginDir,
		},
		Invoke: &it.Invoke{Path: []string{DefaultPluginDir}},
		Cad:    getCadClient(),
	}, nil
}

// ParseCNIConf parses input configuration file and returns
// Genie's native NetConf object.
func ParseCNIConf(confData []byte) (*utils.GenieConf, error) {
	// Unmarshall the network config, and perform validation
	conf := utils.GenieConf{}
	if err := json.Unmarshal(confData, &conf); err != nil {
		return &conf, fmt.Errorf("failed to load netconf: %v", err)
	}
	return &conf, nil
}

// AddPodNetwork adds pod networking. It has logic to parse each pod
// definition's annotations. It looks for container networking solutions (CNS)
// types passed as annotation in pod defintion. For every CNS types, it talks
// to corresponding CNS object and fetches an IP from it's IPAM.
// It also applies the IP as ethX inside the pod.
func (gc *GenieController) AddPodNetwork(cniArgs *utils.CNIArgs, conf *utils.GenieConf) (types.Result, error) {
	k8sArgs, err := loadArgs(cniArgs)
	if err != nil {
		return nil, fmt.Errorf("CNI Genie internal error at loadArgs: %v", err)
	}
	_, _, err = getIdentifiers(cniArgs, k8sArgs)
	if err != nil {
		return nil, fmt.Errorf("CNI Genie internal error at getIdentifiers: %v", err)
	}

	// Get pod annotations
	podAnnot, err := gc.getPodAnnotationsForCNI(k8sArgs)
	if err != nil {
		return nil, fmt.Errorf("Error getting annotations for pod (%s:%s): %v", k8sArgs.K8S_POD_NAMESPACE, k8sArgs.K8S_POD_NAME, err)
	}

	err = gc.Cfg.LoadConfFiles()
	if err != nil {
		return nil, fmt.Errorf("Error loading configuration files from net dir (%s): %v", gc.Cfg.NetDir, err)
	}
	fmt.Fprintf(os.Stderr, "CNI Genie Found configuration files in %s: %v\n", gc.Cfg.NetDir, gc.Cfg.Files)
	//fmt.Println("Files: ", gc.Cfg.Files)
	// parse pod annotations for cns types
	// eg:
	//    cni: "canal,weave"
	var pluginInfoList []*utils.PluginInfo
	var setStatus SetStatus
	var statusAnnot string
	if networkCrdAnnot, ok := podAnnot[NetworkAttachmentDefinitionAnnot]; ok {
		pluginInfoList, err = gc.parseNetAttachDefAnnot(networkCrdAnnot, k8sArgs)
		if err != nil {
			return nil, fmt.Errorf("CNI Genie error at parseNetAttachDefAnnot: %v", err)
		}
		setStatus = setNetAttachStatus
		statusAnnot = NetworkAttachmentStatusAnnot
	} else {
		pluginInfoList, err = gc.parseCNIAnnotations(podAnnot, k8sArgs, conf)
		if err != nil {
			return nil, fmt.Errorf("CNI Genie error at ParsePodAnnotations: %v", err)
		}
		if len(pluginInfoList) > 1 {
			setStatus = setGenieStatus
			statusAnnot = MultiIPPreferencesAnnotation
		}
	}

	result, status, err := gc.addNetwork(pluginInfoList, cniArgs, setStatus)
	if err != nil {
		return nil, err
	}

	var bytes []byte
	if status != nil {
		bytes = getStatusBytes(status)
	}

	if bytes != nil {
		err = gc.UpdatePodDefinition(statusAnnot, bytes, k8sArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie error while setting pod status(%v): %v:\n", string(bytes), err)
		}
	}

	return result, nil
}

// DeletePodNetwork deletes pod networking. It has logic to parse each pod
// definition's annotations. It looks for container networking solutions (CNS)
// types passed as annotation in pod defintion. For every CNS types, it talks
// to corresponding CNS object and releases an IP from it's IPAM.
func (gc *GenieController) DeletePodNetwork(cniArgs *utils.CNIArgs, conf *utils.GenieConf) error {
	k8sArgs, err := loadArgs(cniArgs)
	if err != nil {
		return fmt.Errorf("CNI Genie internal error at loadArgs: %v", err)
	}
	_, _, err = getIdentifiers(cniArgs, k8sArgs)
	if err != nil {
		return fmt.Errorf("CNI Genie internal error at getIdentifiers: %v", err)
	}

	podAnnot, err := gc.getPodAnnotationsForCNI(k8sArgs)
	if err != nil {
		if err_nopod_novar == err.Error() {
			//Incase of pos container delete, getting pod info will fail. So return success in this case
			//to ensure complete cleanup of pos container
			fmt.Fprintf(os.Stderr, "Pod annotations not found during pod delete, proceeding to delete pod")
			return nil
		}
		return fmt.Errorf("Error getting annotations for pod (%s:%s): %v", k8sArgs.K8S_POD_NAMESPACE, k8sArgs.K8S_POD_NAME, err)
	}

	err = gc.Cfg.LoadConfFiles()
	if err != nil {
		return fmt.Errorf("Error loading configuration files from net dir (%s): %v", gc.Cfg.NetDir, err)
	}
	// parse pod annotations for cns types
	// eg:
	//    cni: "canal,weave"
	var pluginInfoList []*utils.PluginInfo
	if networkCrdAnnot, ok := podAnnot[NetworkAttachmentDefinitionAnnot]; ok {
		pluginInfoList, err = gc.parseNetAttachDefAnnot(networkCrdAnnot, k8sArgs)
		if err != nil {
			return fmt.Errorf("CNI Genie error at parseNetAttachDefAnnot: %v", err)
		}
	} else {
		pluginInfoList, err = gc.parseCNIAnnotations(podAnnot, k8sArgs, conf)
		if err != nil {
			return fmt.Errorf("CNI Genie error at ParsePodAnnotations: %v", err)
		}
	}

	return gc.deleteNetwork(pluginInfoList, cniArgs)
}

func getReservedIfnames(pluginElems []*utils.PluginInfo) (map[int64]bool, error) {
	reserved := make(map[int64]bool)
	req := make(map[string]int)
	l := len(DefaultIfNamePrefix)
	for i := range pluginElems {
		if ifName := pluginElems[i].IfName; ifName != "" {
			if req[ifName] == 0 {
				req[ifName]++
			} else {
				return nil, fmt.Errorf("Repeated request for same interface name: %s", ifName)
			}
			if index := strings.Index(ifName, DefaultIfNamePrefix); index == 0 {
				i, err := strconv.ParseInt(ifName[index+l:], 10, 64)
				if err != nil {
					continue
				}
				reserved[i] = true
			}
		}
	}
	return reserved, nil
}

func getIntfName(intfName string, reserved map[int64]bool, curr int) (string, int) {
	if intfName == "" {
		for curr++; true == reserved[int64(curr)]; curr++ {
		}
		intfName = DefaultIfNamePrefix + fmt.Sprintf("%d", curr)
	}

	return intfName, curr
}

func parseResult(setStatus SetStatus, ch chan sendCh) (types.Result, interface{}) {
	var status interface{}
	var endResult *current.Result
	for r := range ch {
		currentResult, err := current.NewResultFromResult(r.res)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie error converting result to current version for plugin %s: %v\n", r.name, err)
			continue
		}
		endResult, err = mergeWithResult(currentResult, endResult)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie error merging current result for plugin %s with end result: %v\n", r.res, err)
			continue
		}
		if setStatus != nil {
			status = setStatus(*currentResult, r.name, r.ifName, status)
		}
	}
	return endResult, interface{}(status)
}

func (gc *GenieController) fillMandatoryCNIPara(config *libcni.NetworkConfigList) {
	// This function is used to check if any cni mandatory parameters are missing. If missing, they will be filled
	//with default parameters

	// Check for cni version field
	if config.CNIVersion == "" {
		fmt.Fprintf(os.Stderr, "CNI Version is missing, filling with default value: %v\n", DefaultCNIVersion)
		config.CNIVersion = DefaultCNIVersion
	}

	return
}

func (gc *GenieController) addNetwork(pluginElements []*utils.PluginInfo, cniArgs *utils.CNIArgs, setStatus SetStatus) (types.Result, interface{}, error) {
	// Collect the result in this variable - this is ultimately what gets "returned" by this function by printing
	// it to stdout.
	var endResult types.Result
	var result types.Result

	reservedIfNames, err := getReservedIfnames(pluginElements)
	if err != nil {
		return nil, nil, err
	}

	var currIndex int = -1
	var status interface{}
	ch := make(chan sendCh, len(pluginElements))

	var wg sync.WaitGroup
	go func(SetStatus, chan sendCh) {
		wg.Add(1)
		defer wg.Done()
		endResult, status = parseResult(setStatus, ch)
	}(setStatus, ch)

	var i int
	var pluginElement *utils.PluginInfo
	for i, pluginElement = range pluginElements {
		pluginElement.IfName, currIndex = getIntfName(pluginElement.IfName, reservedIfNames, currIndex)
		fmt.Fprintf(os.Stderr, "CNI Genie adding network for plugin element: %+v\n", *pluginElement)
		// fetches an IP from corresponding CNS IPAM and returns result object
		result, err = gc.delegateAddNetwork(pluginElement, cniArgs)
		fmt.Fprintf(os.Stderr, "CNI Genie addNetwork (%s) err: %v; result: %v\n", pluginElement.PluginName, err, result)
		if err != nil {
			break
		}

		if pluginElement.ValidateRes != nil {
			err = pluginElement.ValidateRes(result, pluginElement.ValidationParams)
			if err != nil {
				break
			}
		}
		ch <- sendCh{pluginElement.PluginName, pluginElement.IfName, result}
		i++
	}
	close(ch)
	if i < len(pluginElements) {
		_ = gc.deleteNetwork(pluginElements[:i], cniArgs)
		return nil, nil, err
	}

	wg.Wait()
	return endResult, status, nil
}

// addNetwork is a core function that delegates call to pull IP from a Container Networking Solution (CNI Plugin)
func (gc *GenieController) delegateAddNetwork(pluginInfo *utils.PluginInfo, cniArgs *utils.CNIArgs) (types.Result, error) {
	if err := os.Unsetenv("CNI_IFNAME"); err != nil {
		fmt.Fprintf(os.Stderr, "CNI Genie Error while unsetting env variable CNI_IFNAME: %v\n", err)
	}
	if err := os.Unsetenv("CNI_ARGS"); err != nil {
		fmt.Fprintf(os.Stderr, "CNI Genie Error while unsetting env variable CNI_Args: %v\n", err)
	}
	rtConf, err := runtimeConf(cniArgs, pluginInfo.IfName, pluginInfo.OptionalArgs)
	if err != nil {
		return nil, fmt.Errorf("Error generating runtime conf: %v", err)
	}
	fmt.Fprintf(os.Stderr, "CNI Genie runtime conf for plugin (%s): %v\n", pluginInfo.PluginName, *rtConf)

	gc.fillMandatoryCNIPara(pluginInfo.Config)

	res, err := gc.Invoke.InvokeExecAdd(pluginInfo.Config, rtConf)
	if err != nil {
		return nil, fmt.Errorf("Error from cni: %v", err)
	}

	return res, nil
}

// deleteNetwork is a core function that delegates call to release IP from a Container Networking Solution (CNI Plugin)
func (gc *GenieController) deleteNetwork(pluginElements []*utils.PluginInfo, cniArgs *utils.CNIArgs) error {
	reservedIfNames, _ := getReservedIfnames(pluginElements)
	currIndex := -1
	var cnierr error
	for i := len(pluginElements) - 1; i >= 0; i-- {
		pluginElement := pluginElements[i]
		pluginElement.IfName, currIndex = getIntfName(pluginElement.IfName, reservedIfNames, currIndex)
		fmt.Fprintf(os.Stderr, "CNI Genie deleting network for plugin %s\n", pluginElement.PluginName)
		// releases an IP from corresponding CNS IPAM and returns error if any exception
		err := gc.delegateDelNetwork(pluginElement, cniArgs)
		if err != nil {
			cnierr = err
			fmt.Fprintf(os.Stderr, "CNI Genie Error while deleting network (%s): %v\n", pluginElement.PluginName, err)
			continue
		}
	}

	if cnierr != nil {
		return cnierr
	}

	fmt.Fprintln(os.Stderr, "CNI Genie deleteNetwork successful")
	return nil
}

func (gc *GenieController) delegateDelNetwork(pluginInfo *utils.PluginInfo, cniArgs *utils.CNIArgs) error {
	rtConf, err := runtimeConf(cniArgs, pluginInfo.IfName, pluginInfo.OptionalArgs)
	if err != nil {
		return fmt.Errorf("CNI Genie couldn't convert cniArgs to RuntimeConf: %v", err)
	}

	fmt.Fprintf(os.Stderr, "CNI Genie runtime conf for plugin (%s): %v\n", pluginInfo.PluginName, *rtConf)

	gc.fillMandatoryCNIPara(pluginInfo.Config)

	err = gc.Invoke.InvokeExecDel(pluginInfo.Config, rtConf)
	if err != nil {
		return fmt.Errorf("Error from cni: %v", err)
	}

	return nil
}

// UpdatePodDefinition updates the pod definition with the given annotation.
func (gc *GenieController) UpdatePodDefinition(statusAnnot string, status []byte, k8sArgs *utils.K8sArgs) error {
	annot := fmt.Sprintf(
		`{"metadata":{"annotations":{"%s":%s}}}`, statusAnnot, strconv.Quote(string(status)))

	fmt.Fprintf(os.Stderr, "CNI Genie pod.Annotations[%s] after = %s\n", statusAnnot, annot)
	_, err := gc.Kc.CoreV1().Pods(string(k8sArgs.K8S_POD_NAMESPACE)).Patch(string(k8sArgs.K8S_POD_NAME), api.StrategicMergePatchType, []byte(annot))
	if err != nil {
		return fmt.Errorf("CNI Genie Error updating pod = %s", err)
	}
	return nil
}

// GetPodDefinition gets pod definition through k8s api server
func GetPodDefinition(client *kubernetes.Clientset, podNamespace string, podName string) (*v1.Pod, error) {
	pod, err := client.CoreV1().Pods(podNamespace).Get(fmt.Sprintf("%s", podName), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pod, nil
}

func (gc *GenieController) getPodAnnotationsForCNI(k8sArgs *utils.K8sArgs) (map[string]string, error) {
	annot, err := gc.getK8sPodAnnotations(k8sArgs)
	if err != nil {
		args := k8sArgs.K8S_ANNOT
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "CNI Genie no env var and no pod")
			return annot, errors.New(err_nopod_novar)
		}
		fmt.Fprintf(os.Stderr, "CNI Genie env  annot val: %s\n", args)
		envAnnot := map[string]string{}
		errEnv := json.Unmarshal([]byte(args), &envAnnot)
		if errEnv != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie error getting annotations from pod: `%v` and Error Using annotations from ENV: `%v`\n", err, errEnv)
			return annot, err
		}
		annot = envAnnot
		fmt.Fprintf(os.Stderr, "CNI Genie error getting annotations from pod: %v. Using annotations from ENV: annot= %v\n", err, annot)
	}
	fmt.Fprintf(os.Stderr, "CNI Genie annot= [%s]\n", annot)

	return annot, err

}

// ParsePodAnnotationsForNetworks does following tasks
// - get pod definition
// - parses annotation section for "networks"
// - Returns string
func (gc *GenieController) parsePodAnnotationsForNetworks(k8sArgs *utils.K8sArgs) string {
	annot, _ := gc.getK8sPodAnnotations(k8sArgs)
	networks := annot["networks"]
	return networks
}

// List all configuration files in the given directory with specified extensions
func getConfFiles(dir string) ([]string, error) {
	files, err := libcni.ConfFiles(dir, []string{".conf", ".conflist"})
	if err != nil {
		return nil, fmt.Errorf("Error listing configuration files in %s: %v", dir, err)
	}
	fmt.Fprintf(os.Stderr, "CNI Genie files: %v\n", files)
	return files, err
}

// Gets all the info related to plugin using respective plugin config files
func (gc *GenieController) getPluginInfo(plugins []string) ([]*utils.PluginInfo, error) {
	pluginInfoList := make([]*utils.PluginInfo, len(plugins))
	pluginMap := make(map[string]map[bool][]int)
	ifNameMap := make(map[int]string)
	for i := range plugins {
		pluginName := strings.TrimSpace(plugins[i])
		ifName := ""
		if true == strings.Contains(pluginName, utils.IfNameDelimiter) {
			netNIfName := strings.Split(pluginName, utils.IfNameDelimiter)
			pluginName = strings.TrimSpace(netNIfName[0])
			ifName = strings.TrimSpace(netNIfName[1])
		}

		if pluginMap[pluginName] == nil {
			pluginMap[pluginName] = map[bool][]int{false: {i + 1}}
		} else {
			pluginMap[pluginName][false] = append(pluginMap[pluginName][false], i+1)
		}
		ifNameMap[i] = ifName
	}
	fmt.Fprintf(os.Stderr, "CNI Genie plugin map: %+v\n", pluginMap)
	for _, file := range gc.Cfg.Files {
		// Parse file name and check whether it matches any of the requested plugins
		// In conf file name, the plugin name should be followed by a '.' and
		// should be preceded by either '-' or nothing
		pluginName := strings.TrimSpace(file[strings.LastIndex(file, "/")+1 : strings.LastIndex(file, ".")])
		if i := strings.Index(pluginName, "-"); i > 0 {
			pluginName = pluginName[i+1:]
		}

		if pluginMap[pluginName] != nil {
			indices := pluginMap[pluginName][false]
			pluginMap[pluginName][true] = indices
			config, err := gc.Cfg.ParseCNIConfFromFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "CNI Genie Error getting CNI config from conf file (%s) for user requested plugin (%s): %v\n", file, pluginName, err)
				continue
			}
			fmt.Fprintf(os.Stderr, "CNI Genie found configuration file (%s) for plugin %s\n", file, pluginName)
			for _, index := range indices {
				pluginInfoList[index-1] = &utils.PluginInfo{
					PluginName: pluginName,
					Config:     config,
					IfName:     ifNameMap[index-1],
				}
			}
			delete(pluginMap, pluginName)
		}
		if len(pluginMap) == 0 {
			break
		}
	}

	// pluginMap is still not empty means, for one or more plugins,
	// either we did not find a valid conf file or we do not support
	// the plugin or we support the plugin, but we need to place a
	// default conf file for the plugin
	for plugin, v := range pluginMap {
		if _, ok := v[true]; ok {
			return nil, fmt.Errorf("No valid configuration file present for plugin %s", plugin)
		}
		config, err := gc.generateConf(plugin)
		if err != nil {
			return nil, err
		}
		for _, index := range v[false] {
			pluginInfoList[index-1] = &utils.PluginInfo{
				PluginName: plugin,
				Config:     config,
			}
		}
	}

	return pluginInfoList, nil
}

func (gc *GenieController) loadPluginConfig(plugin string) (*libcni.NetworkConfigList, error) {
	if plugin = strings.TrimSpace(plugin); plugin == "" {
		return nil, fmt.Errorf("Plugin name is empty")
	}

	found := false
	for _, file := range gc.Cfg.Files {
		name := file[strings.LastIndex(file, "/")+1 : strings.LastIndex(file, ".")]
		if i := strings.Index(name, "-"); i > 0 {
			name = name[i+1:]
		}
		if name == plugin {
			found = true
			config, err := gc.Cfg.ParseCNIConfFromFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "CNI Genie Error getting CNI config from conf file (%s) for user requested plugin (%s): %v\n", file, plugin, err)
				continue
			}
			return config, nil
		}
	}

	if found == true {
		return nil, fmt.Errorf("No valid configuration file present for plugin %s", plugin)
	}
	config, err := gc.generateConf(plugin)
	if err != nil {
		return nil, err
	}

	return config, nil
}

//  parseCNIAnnotations parses pod yaml defintion for "cni" annotations.
func (gc *GenieController) parseCNIAnnotations(annot map[string]string, k8sArgs *utils.K8sArgs, conf *utils.GenieConf) ([]*utils.PluginInfo, error) {
	var finalPluginInfos []*utils.PluginInfo
	var err error

	_, annotExists := annot["cni"]

	if !annotExists {
		fmt.Fprintf(os.Stderr, "CNI Genie no annotations is given! Using default plugins \n")
		finalPluginInfos, err = gc.handleNoCniCase(conf)
		if err != nil {
			return nil, err
		}
	} else if strings.TrimSpace(annot["cni"]) != "" {
		cniAnnots := strings.Split(annot["cni"], ",")
		finalPluginInfos, err = gc.getPluginInfo(cniAnnots)
		if err != nil {
			return nil, err
		}
	} else if networksAnnot := gc.parsePodAnnotationsForNetworks(k8sArgs); networksAnnot != "" {
		fmt.Fprintf(os.Stderr, "CNI Genie networks annotation passed\n")

		var err error

		finalPluginInfos, err = gc.getPluginInfoFromNwAnnot(strings.TrimSpace(annot["networks"]), string(k8sArgs.K8S_POD_NAMESPACE))
		if err != nil {
			return finalPluginInfos, fmt.Errorf("CNI Genie GetPluginInfoFromNwAnnot err= %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "CNI Genie Inside no cni annotation, calling cAdvisor client to retrieve ideal network solution\n")
		cns, err := gc.GetCNSOrderByNetworkBandwith(conf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie GetCNSOrderByNetworkBandwith err= %v\n", err)
			return finalPluginInfos, fmt.Errorf("CNI Genie failed to retrieve CNS list from cAdvisor = %v", err)
		}
		fmt.Fprintf(os.Stderr, "CNI Genie cns= %v\n", cns)

		cni := fmt.Sprintf(`{"metadata":{"annotations":{"cni":"%s"}}}`, cns)
		_, err = gc.Kc.PatchPod(string(k8sArgs.K8S_POD_NAME), string(k8sArgs.K8S_POD_NAMESPACE), api.StrategicMergePatchType, []byte(cni))
		if err != nil {
			return finalPluginInfos, fmt.Errorf("CNI Genie Error updating pod = %s", err)
		}

		finalPluginInfos, err = gc.getPluginInfo([]string{cns})
		if err != nil {
			return nil, err
		}
	}

	fmt.Fprintf(os.Stderr, "CNI Genie length of finalPluginInfos= %v\n", len(finalPluginInfos))
	return finalPluginInfos, nil
}

// checkPluginBinary checks for existence of plugin binary file
func checkPluginBinary(cniName string) error {
	binaries, err := ioutil.ReadDir(DefaultPluginDir)
	if err != nil {
		return fmt.Errorf("Error while checking binary file for plugin %s: %v", cniName, err)
	}

	for _, bin := range binaries {
		if true == strings.Contains(bin.Name(), cniName) {
			return nil
		}
	}
	return fmt.Errorf("Corresponding binary for user requested plugin (%s) is not present in plugin directory (%s)", cniName, DefaultPluginDir)
}

// createConfIfBinaryExists checks for the binary file for a cni type and creates the conf if binary exists
func (gc *GenieController) createConfIfBinaryExists(cniName string) (*libcni.NetworkConfigList, error) {
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
		return nil, fmt.Errorf("Configuration file is missing from cni directory (%s) for user requested plugin: %s", DefaultNetDir, cniName)
	}

	confBytes, err := json.MarshalIndent(pluginObj, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("Error while marshalling configuration object for plugin %s: %v", cniName, err)
	}

	err = gc.Cfg.CreateConfFile(cniName, confBytes)
	if err != nil {
		return nil, fmt.Errorf("Error placing conf file for plugin %s: %v", cniName, err)
	}

	confList, err := gc.Cfg.ConfListFromConfBytes(confBytes)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "CNI Genie Placed default conf file for cni type %s.\n", cniName)

	return confList, nil
}

func insertSubnet(conf map[string]interface{}, subnet string) {
	ipam := make(map[string]interface{})

	if conf["ipam"] != nil {
		ipam = conf["ipam"].(map[string]interface{})
	}
	ipam["subnet"] = subnet
	conf["ipam"] = ipam
}

func useCustomSubnet(confdata []byte, subnet string) ([]byte, error) {
	conf := make(map[string]interface{})
	err := json.Unmarshal([]byte(confdata), &conf)
	if err != nil {
		return nil, fmt.Errorf("Error Unmarshalling confdata: %v", err)
	}

	// If it is a conflist
	if conf["plugins"] != nil {
		// Considering the 0th element in the plugin array as the plugin configuration
		insertSubnet(conf["plugins"].([]interface{})[0].(map[string]interface{}), subnet)
	} else {
		insertSubnet(conf, subnet)
	}

	confbytes, err := json.Marshal(&conf)
	if err != nil {
		return nil, fmt.Errorf("Error Marshalling confdata: %v", err)
	}

	return confbytes, nil
}

func (gc *GenieController) generateConf(cniName string) (*libcni.NetworkConfigList, error) {
	supportedPlugins := strings.Split(SupportedPlugins, ",")
	var cnt int
	for _, plugin := range supportedPlugins {
		if cniName == strings.TrimSpace(plugin) {
			break
		}
		cnt++
	}
	if cnt >= len(supportedPlugins) {
		return nil, fmt.Errorf("User requested for unsupported plugin type %s. Only supported are %s", cniName, SupportedPlugins)
	}

	return gc.createConfIfBinaryExists(cniName)
}

func loadArgs(cniArgs *utils.CNIArgs) (*utils.K8sArgs, error) {
	k8sArgs := &utils.K8sArgs{}
	err := types.LoadArgs(cniArgs.Args, k8sArgs)
	if err != nil {
		return k8sArgs, err
	}
	return k8sArgs, nil
}

func getIdentifiers(cniArgs *utils.CNIArgs, k8sArgs *utils.K8sArgs) (workloadID string, orchestratorID string, err error) {
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

func (gc *GenieController) getK8sPodAnnotations(k8sArgs *utils.K8sArgs) (map[string]string, error) {
	pod, err := gc.Kc.GetPod(string(k8sArgs.K8S_POD_NAME), string(k8sArgs.K8S_POD_NAMESPACE))
	if err != nil {
		return nil, err
	}
	return pod.Annotations, nil
}

func runtimeConf(cniArgs *utils.CNIArgs, iface string, optionalArgs map[string]string) (*libcni.RuntimeConf, error) {
	k8sArgs, err := loadArgs(cniArgs)
	if err != nil {
		return nil, err
	}
	args := [][2]string{}
	if k8sArgs.IgnoreUnknown {
		args = append(args, [2]string{"IgnoreUnknown", "1"})
	}
	if string(k8sArgs.K8S_POD_NAMESPACE) != "" {
		args = append(args, [2]string{"K8S_POD_NAMESPACE", string(k8sArgs.K8S_POD_NAMESPACE)})
	}
	if string(k8sArgs.K8S_POD_NAME) != "" {
		args = append(args, [2]string{"K8S_POD_NAME", string(k8sArgs.K8S_POD_NAME)})
	}
	if string(k8sArgs.K8S_POD_INFRA_CONTAINER_ID) != "" {
		args = append(args, [2]string{"K8S_POD_INFRA_CONTAINER_ID", string(k8sArgs.K8S_POD_INFRA_CONTAINER_ID)})
	}

	args = append(args, setOptionalArgs(optionalArgs)...)

	return &libcni.RuntimeConf{
		ContainerID: cniArgs.ContainerID,
		NetNS:       cniArgs.Netns,
		IfName:      iface,
		Args:        args}, nil
}

func setOptionalArgs(optionalParam map[string]string) [][2]string {
	args := [][2]string{}
	if optionalParam["ips"] != "" {
		args = append(args, [2]string{"IP", optionalParam["ips"]})
	}
	if optionalParam["mac"] != "" {
		args = append(args, [2]string{"MAC", optionalParam["mac"]})
	}
	return args
}

// Function used to select plugin when cni annotation is not provided
func (gc *GenieController) handleNoCniCase(conf *utils.GenieConf) ([]*utils.PluginInfo, error) {
	var pluginInfoList []*utils.PluginInfo
	var err error

	//If no default plugin is mentioned, select first valid conf/conflist file in cni dir as kubelet does
	if conf.DefaultPlugin == "" {
		config, err := gc.getClusterNetwork(gc.Cfg.NetDir)
		if err != nil {
			return nil, fmt.Errorf("Failed to get default plugin: %v", err)
		}

		fmt.Fprintf(os.Stderr, "CNI Genie no default plugin provided, selected plugin: %s\n", config.Plugins[0].Network.Type)
		pluginInfoList = append(pluginInfoList, &utils.PluginInfo{PluginName: config.Plugins[0].Network.Type, Config: config, IfName: DefaultIfNamePrefix + "0"})
	} else {
		//Use default plugin specified
		plugins := strings.Split(conf.DefaultPlugin, ",")
		pluginInfoList, err = gc.getPluginInfo(plugins)
		if err != nil {
			return nil, err
		}
	}
	return pluginInfoList, nil
}

func mergeWithResult(src *current.Result, dst *current.Result) (*current.Result, error) {
	err := updateRoutes(src)
	if err != nil {
		return nil, fmt.Errorf("Routes update failed: %v", err)
	}
	err = fixInterfaces(src)
	if err != nil {
		return nil, fmt.Errorf("Failed to fix interfaces: %v", err)
	}

	if dst == nil {
		return src, nil
	}

	ifacesLength := len(dst.Interfaces)

	for _, iface := range src.Interfaces {
		dst.Interfaces = append(dst.Interfaces, iface)
	}
	for _, ip := range src.IPs {
		if ip.Interface != nil && *(ip.Interface) != -1 {
			ip.Interface = current.Int(*(ip.Interface) + ifacesLength)
		}
		dst.IPs = append(dst.IPs, ip)
	}
	for _, route := range src.Routes {
		dst.Routes = append(dst.Routes, route)
	}

	for _, ns := range src.DNS.Nameservers {
		dst.DNS.Nameservers = append(dst.DNS.Nameservers, ns)
	}
	for _, s := range src.DNS.Search {
		dst.DNS.Search = append(dst.DNS.Search, s)
	}
	for _, opt := range src.DNS.Options {
		dst.DNS.Options = append(dst.DNS.Options, opt)
	}
	// TODO: what about DNS.domain?
	return dst, nil
}

// updateRoutes changes nil gateway set in a route to a gateway from IPConfig
// nil gw in route means default gw from result. When merging results from
// many results default gw may be set from another CNI network. This may lead to
// wrong routes.
func updateRoutes(result *current.Result) error {
	if len(result.Routes) == 0 {
		return nil
	}

	var gw net.IP
	for _, ip := range result.IPs {
		if ip.Gateway != nil {
			gw = ip.Gateway
			break
		}
	}

	for _, route := range result.Routes {
		if route.GW == nil {
			if gw == nil {
				return fmt.Errorf("Couldn't find gw in result %v", result)
			}
			route.GW = gw
		}
	}
	return nil
}

// fixInterfaces fixes bad result returned by CNI plugin
// some plugins(for example calico) return empty Interfaces list but
// in IPConfig sets Interface index to 0. In such case it should be nil
func fixInterfaces(result *current.Result) error {
	if len(result.Interfaces) == 0 {
		for _, ip := range result.IPs {
			ip.Interface = nil
		}
	}
	return nil
}
