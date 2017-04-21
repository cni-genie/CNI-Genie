//
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
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
	. "github.com/projectcalico/cni-plugin/utils"
	"github.com/projectcalico/cni-plugin/utils"
	"github.com/containernetworking/cni/pkg/ipam"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	log "github.com/Sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
	"strconv"
)

var hostname string

func init() {
	// This ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()

	hostname, _ = os.Hostname()
}

func cmdAdd(args *skel.CmdArgs) error {
	// Unmarshall the network config, and perform validation
	conf := NetConf{}
	if err := json.Unmarshal(args.StdinData, &conf); err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}

	workload, _, err := getIdentifiers(args)
	if err != nil {
		return err
	}

	// Allow the hostname to be overridden by the network config
	if conf.Hostname != "" {
		hostname = conf.Hostname
	}

	logger := createContextLogger(workload)
	client, err := newK8sClient(conf, logger)
	if err != nil {
		return err
	}
	k8sArgs := K8sArgs{}
	err = types.LoadArgs(args.Args, &k8sArgs)
	if err != nil {
		return err
	}
	annot := make(map[string]string)
	_, annot, err = getK8sLabelsAnnotations(client, k8sArgs)
	annotStringArray := strings.Split(annot["cni"], ",")
	fmt.Fprintf(os.Stderr, "CNI Genie annot= %s\n", annot)

	// Collect the result in this variable - this is ultimately what gets "returned" by this function by printing
	// it to stdout.
	var result types.Result

	i := 0
	for i < len(annotStringArray) {
		switch annotStringArray[i] {
		case "weave":
			conf.IPAM.Type = "weave-ipam"
			conf.Type = "weave-net"
			args.StdinData,_ = json.Marshal(&conf)
			if os.Setenv("CNI_IFNAME", "eth" + strconv.Itoa(i)) != nil {
				fmt.Fprintf(os.Stderr, "CNI_IFNAME Error\n")
			}
			result, err = ipam.ExecAdd("weave-net", args.StdinData)
			if err != nil {
				return err
			}
		case "calico":
			conf.IPAM.Type = "calico-ipam"
			conf.Type = "calico"
			args.StdinData,_ = json.Marshal(&conf)
			if os.Setenv("CNI_IFNAME", "eth" + strconv.Itoa(i)) != nil {
				fmt.Fprintf(os.Stderr, "CNI_IFNAME Error\n")
			}
			result, err = ipam.ExecAdd("calico", args.StdinData)
			if err != nil {
				return err
			}
		case "canal":
			conf.Type = "flannel"
			conf.Delegate.DelegateType = "calico"
			conf.Delegate.EtcdEndpoints = conf.EtcdEndpoints
			conf.Delegate.LogLevel = conf.LogLevel
			conf.Delegate.Policy = conf.Policy
			conf.Delegate.Kubernetes = conf.Kubernetes
			args.StdinData, _ = json.Marshal(&conf)
			if os.Setenv("CNI_IFNAME", "eth" + strconv.Itoa(i)) != nil {
				fmt.Fprintf(os.Stderr, "CNI_IFNAME Error\n")
			}
			result, err = ipam.ExecAdd("flannel", args.StdinData)
			if err != nil {
				return err
			}
		}
		i += 1
	}

	fmt.Fprintf(os.Stderr, "CNI Genie result= %s\n", result)
	return types.PrintResult(result,VERSION)
}

func cmdDel(args *skel.CmdArgs) error {
	conf := NetConf{}
	if err := json.Unmarshal(args.StdinData, &conf); err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}

	fmt.Fprintf(os.Stderr, "CNI Genie releasing IP address\n")

	workload, _, err := getIdentifiers(args)
	if err != nil {
		return err
	}

	// Allow the hostname to be overridden by the network config
	if conf.Hostname != "" {
		hostname = conf.Hostname
	}

	logger := createContextLogger(workload)
	client, err := newK8sClient(conf, logger)
	if err != nil {
		return err
	}
	k8sArgs := K8sArgs{}
	err = types.LoadArgs(args.Args, &k8sArgs)
	if err != nil {
		return err
	}
	annot := make(map[string]string)
	_, annot, err = getK8sLabelsAnnotations(client, k8sArgs)
	annotStringArray := strings.Split(annot["cni"], ",")
	fmt.Fprintf(os.Stderr, "CNI Genie annot= %s\n", annot)

	// Collect the result in this variable - this is ultimately what gets "returned" by this function by printing
	// it to stdout.
	var ipamErr error

	i := 0
	for i < len(annotStringArray) {
		switch annotStringArray[i] {
		case "weave":
			conf.IPAM.Type = "weave-ipam"
			conf.Type = "weave-net"
			args.StdinData, _ = json.Marshal(&conf)
			if os.Setenv("CNI_IFNAME", "eth" + strconv.Itoa(i)) != nil {
				fmt.Fprintf(os.Stderr, "CNI_IFNAME Error\n")
			}
			ipamErr := ipam.ExecDel("weave-net", args.StdinData)
			if ipamErr != nil {
				fmt.Fprintf(os.Stderr, "ipamErr= %s\n", ipamErr)
			}
		case "calico":
			conf.IPAM.Type = "calico-ipam"
			conf.Type = "calico"
			args.StdinData, _ = json.Marshal(&conf)
			if os.Setenv("CNI_IFNAME", "eth" + strconv.Itoa(i)) != nil {
				fmt.Fprintf(os.Stderr, "CNI_IFNAME Error\n")
			}
			ipamErr := ipam.ExecDel("calico", args.StdinData)
			if ipamErr != nil {
				fmt.Fprintf(os.Stderr, "ipamErr= %s\n", ipamErr)
			}
		case "canal":
			conf.Type = "flannel"
			conf.Delegate.DelegateType = "calico"
			conf.Delegate.EtcdEndpoints = conf.EtcdEndpoints
			conf.Delegate.LogLevel = conf.LogLevel
			conf.Delegate.Policy = conf.Policy
			conf.Delegate.Kubernetes = conf.Kubernetes
			args.StdinData, _ = json.Marshal(&conf)
			if os.Setenv("CNI_IFNAME", "eth" + strconv.Itoa(i)) != nil {
				fmt.Fprintf(os.Stderr, "CNI_IFNAME Error\n")
			}
			ipamErr := ipam.ExecDel("flannel", args.StdinData)
			if ipamErr != nil {
				fmt.Fprintf(os.Stderr, "ipamErr= %s\n", ipamErr)
			}
		}
		i += 1
	}

	return ipamErr
}

func getK8sLabelsAnnotations(client *kubernetes.Clientset, k8sargs utils.K8sArgs) (map[string]string, map[string]string, error) {
	pod, err := client.Pods(string(k8sargs.K8S_POD_NAMESPACE)).Get(fmt.Sprintf("%s", k8sargs.K8S_POD_NAME), metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	labels := pod.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	labels["calico/k8s_ns"] = fmt.Sprintf("%s", k8sargs.K8S_POD_NAMESPACE)

	return labels, pod.Annotations, nil
}

// Create a logger which always includes common fields
func createContextLogger(workload string) *log.Entry {
	// A common pattern is to re-use fields between logging statements by re-using
	// the logrus.Entry returned from WithFields()
	contextLogger := log.WithFields(log.Fields{
		"Workload": workload,
	})

	return contextLogger
}

func getIdentifiers(args *skel.CmdArgs) (workloadID string, orchestratorID string, err error) {
	// Determine if running under k8s by checking the CNI args
	k8sArgs := K8sArgs{}
	if err = types.LoadArgs(args.Args, &k8sArgs); err != nil {
		return workloadID, orchestratorID, err
	}

	if string(k8sArgs.K8S_POD_NAMESPACE) != "" && string(k8sArgs.K8S_POD_NAME) != "" {
		workloadID = fmt.Sprintf("%s.%s", k8sArgs.K8S_POD_NAMESPACE, k8sArgs.K8S_POD_NAME)
		orchestratorID = "k8s"
	} else {
		workloadID = args.ContainerID
		orchestratorID = "cni"
	}
	return workloadID, orchestratorID, nil
}

func newK8sClient(conf utils.NetConf, logger *log.Entry) (*kubernetes.Clientset, error) {
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

	logger.Debugf("Kubernetes config %v", config)

	// Create the clientset
	return kubernetes.NewForConfig(config)
}

var VERSION = "0.3.1"
var versionInfo     version.PluginInfo

func main() {
	// Display the version on "-v", otherwise just delegate to the skel code.
	// Use a new flag set so as not to conflict with existing libraries which use "flag"
	versionInfo = version.PluginSupports(VERSION)
	skel.PluginMain(cmdAdd, cmdDel,versionInfo)
}
