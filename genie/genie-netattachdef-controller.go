package genie

import (
	"fmt"
	"github.com/Huawei-PaaS/CNI-Genie/networkcrd"
	"github.com/Huawei-PaaS/CNI-Genie/utils"
	"github.com/containernetworking/cni/libcni"
	"k8s.io/client-go/kubernetes"
	"os"
	"strings"
)

func parseNetAttachDefAnnot(annot string, kubeClient *kubernetes.Clientset, k8sArgs utils.K8sArgs, cniDir string) ([]*utils.PluginInfo, error) {
	networks, err := networkcrd.GetNetworkInfo(annot, string(k8sArgs.K8S_POD_NAMESPACE))
	if err != nil {
		return nil, fmt.Errorf("Error parsing network selection annotation: %v", err)
	}
	fmt.Fprintf(os.Stderr, "CNI Genie network elements from network selection annotation: %+v\n", networks)

	var pluginInfoList []*utils.PluginInfo
	for _, netElem := range networks {
		network, err := networkcrd.GetNetworkCRDObject(kubeClient, netElem.Name, netElem.Namespace)
		if err != nil {
			return nil, fmt.Errorf("Error getting network crd object: %v", err)
		}

		pluginInfo := utils.PluginInfo{}
		pluginInfo.Config, err = getNetworkConfig(network, cniDir)
		if err != nil {
			return nil, err
		}

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

func getNetworkConfig(network *networkcrd.NetworkAttachmentDefinition, cniDir string) (*libcni.NetworkConfigList, error) {
	var config *libcni.NetworkConfigList
	var err error
	emptySpec := networkcrd.NetworkAttachmentDefinitionSpec{}
	if network.Spec == emptySpec || network.Spec.Config == "" {
		config, err = networkcrd.GetConfigFromFile(network, cniDir)
		if err != nil {
			return nil, fmt.Errorf("Error extracting plugin configuration from configuration file for net-attach-def object (%s:%s): %v", network.Namespace, network.Name, err)
		}
	} else {
		config, err = networkcrd.GetConfigFromSpec(network)
		if err != nil {
			return nil, fmt.Errorf("Error extracting plugin configuration from object spec for net-attach-def object (%s:%s): %v", network.Namespace, network.Name, err)
		}
	}

	return config, nil
}
