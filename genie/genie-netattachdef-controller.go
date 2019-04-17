package genie

import (
	"fmt"
	"github.com/cni-genie/CNI-Genie/networkcrd"
	"github.com/cni-genie/CNI-Genie/utils"
	"github.com/containernetworking/cni/libcni"
	"os"
	"strings"
)

const (
	GenieConfFile = "00-genie.conf"
)

func (gc *GenieController) parseNetAttachDefAnnot(annot string, k8sArgs *utils.K8sArgs) ([]*utils.PluginInfo, error) {
	var pluginInfoList []*utils.PluginInfo
	config, err := gc.getClusterNetwork(gc.Cfg.NetDir)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "CNI Genie found default network for cluster: %s\n", config.Plugins[0].Network.Type)
	pluginInfoList = append(pluginInfoList, &utils.PluginInfo{PluginName: config.Plugins[0].Network.Type, Config: config, IfName: DefaultIfNamePrefix + "0"})

	networks, err := networkcrd.GetNetworkInfo(annot, string(k8sArgs.K8S_POD_NAMESPACE))
	if err != nil {
		return nil, fmt.Errorf("Error parsing network selection annotation: %v", err)
	}
	fmt.Fprintf(os.Stderr, "CNI Genie network elements from network selection annotation: %+v\n", networks)

	for _, netElem := range networks {
		network, err := networkcrd.GetNetworkCRDObject(gc.Kc, netElem.Name, netElem.Namespace)
		if err != nil {
			return nil, fmt.Errorf("Error getting network crd object: %v", err)
		}

		pluginInfo := utils.PluginInfo{}
		pluginInfo.Config, err = gc.getNetworkConfig(network)
		if err != nil {
			return nil, err
		}

		pluginInfo.OptionalArgs = make(map[string]string, 0)

		if len(netElem.IPs) > 0 {
			pluginInfo.OptionalArgs["ips"] = strings.Join(netElem.IPs, ",")
		}

		if netElem.Mac != "" {
			pluginInfo.OptionalArgs["mac"] = netElem.Mac
		}
		pluginInfo.PluginName = network.Name
		pluginInfo.IfName = netElem.Interface
		pluginInfoList = append(pluginInfoList, &pluginInfo)
	}

	return pluginInfoList, nil
}

func (gc *GenieController) getNetworkConfig(network *networkcrd.NetworkAttachmentDefinition) (*libcni.NetworkConfigList, error) {
	var config *libcni.NetworkConfigList
	var err error
	emptySpec := networkcrd.NetworkAttachmentDefinitionSpec{}
	if network.Spec == emptySpec || network.Spec.Config == "" {
		config, err = networkcrd.GetConfigFromFile(network, gc.Cfg.NetDir)
		if err != nil {
			return nil, fmt.Errorf("Error extracting plugin configuration from configuration file for net-attach-def object (%s:%s): %v", network.Namespace, network.Name, err)
		}
	} else {
		config, err = networkcrd.GetConfigFromSpec(network, gc.Cfg.CNI)
		if err != nil {
			return nil, fmt.Errorf("Error extracting plugin configuration from object spec for net-attach-def object (%s:%s): %v", network.Namespace, network.Name, err)
		}
	}

	return config, nil
}

// getClusterNetwork gets the cluster wide default network based on the configuration
// files present in cni directory. The first (in lexical order) valid config file
// (excluding 00-genie.conf) will be treated as the configuration for cluster wide
// default network. It will be assumed that the corresponding plugin executable is
// is present in the plugin directory and the corresponding plugin service is running
// (if required) in the cluster. If no conf/conflist file (other than 00-genie.conf)
// is present in the directory, then network attachment will fail assuming non-readiness
// of node.
func (gc *GenieController) getClusterNetwork(cniDir string) (*libcni.NetworkConfigList, error) {
	if len(gc.Cfg.Files) <= 1 {
		return nil, fmt.Errorf("No cni plugin has been installed on node.")
	}
	for _, file := range gc.Cfg.Files {
		if strings.Contains(file, GenieConfFile) {
			continue
		}
		config, err := gc.Cfg.ParseCNIConfFromFile(file)
		if err != nil {
			continue
		}
		return config, nil
	}
	return nil, fmt.Errorf("Unable to select default cluster network. No valid configuration file present in cni directory.")
}
