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
	"fmt"
	"sync"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	networklisters "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	clientset "github.com/Huawei-PaaS/CNI-Genie/controllers/logicalnetwork-pkg/client/clientset/versioned"
	networkscheme "github.com/Huawei-PaaS/CNI-Genie/controllers/logicalnetwork-pkg/client/clientset/versioned/scheme"
	informers "github.com/Huawei-PaaS/CNI-Genie/controllers/logicalnetwork-pkg/client/informers/externalversions"
	listers "github.com/Huawei-PaaS/CNI-Genie/controllers/logicalnetwork-pkg/client/listers/network/v1"

	"encoding/json"
	iptables "github.com/Huawei-PaaS/CNI-Genie/controllers/network-policy-controller/iptables"
	. "github.com/Huawei-PaaS/CNI-Genie/utils"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"
	"strings"
	"time"
)

const (
	ControllerAgentName = "network-policy-controller"
	GenieNetworkPolicy  = "genieNetworkPolicy"
)

type NetworkPolicyController struct {
	kubeclientset kubernetes.Interface
	extclientset  clientset.Interface

	networkPoliciesLister networklisters.NetworkPolicyLister
	networkPoliciesSynced cache.InformerSynced
	logicalNwLister       listers.LogicalNetworkLister
	logicalNwSynced       cache.InformerSynced

	npcWorkqueue workqueue.RateLimitingInterface
	recorder     record.EventRecorder

	iptable iptables.IpTables
	mutex   sync.Mutex
}

// NewNpcController returns a new network policy controller
func NewNpcController(
	kubeclientset kubernetes.Interface,
	extclientset clientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	externalObjInformerFactory informers.SharedInformerFactory) *NetworkPolicyController {

	networkPolicyInformer := kubeInformerFactory.Networking().V1().NetworkPolicies()
	logicalNwInformer := externalObjInformerFactory.Alpha().V1().LogicalNetworks()

	networkscheme.AddToScheme(scheme.Scheme)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: ControllerAgentName})

	npcController := &NetworkPolicyController{
		kubeclientset:         kubeclientset,
		extclientset:          extclientset,
		networkPoliciesLister: networkPolicyInformer.Lister(),
		networkPoliciesSynced: networkPolicyInformer.Informer().HasSynced,
		logicalNwLister:       logicalNwInformer.Lister(),
		logicalNwSynced:       logicalNwInformer.Informer().HasSynced,
		npcWorkqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "npc"),
		recorder:              recorder,
	}

	logicalNwInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    npcController.addLogicalNetwork,
		UpdateFunc: npcController.updateLogicalNetwork,
		DeleteFunc: npcController.deleteLogicalNetwork,
	})

	networkPolicyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    npcController.addPolicy,
		UpdateFunc: npcController.updatePolicy,
		DeleteFunc: npcController.deletePolicy,
	})

	return npcController
}

func (npc *NetworkPolicyController) addLogicalNetwork(obj interface{}) {
	l := obj.(*LogicalNetwork)
	npc.enqueueLogicalNetwork(l, "ADD", "")
}

func (npc *NetworkPolicyController) updateLogicalNetwork(old, cur interface{}) {
	oldLn := old.(*LogicalNetwork)
	newLn := cur.(*LogicalNetwork)

	if oldLn.ResourceVersion == newLn.ResourceVersion {
		return
	}

	if oldLn.Spec.SubSubnet != newLn.Spec.SubSubnet {
		npc.enqueueLogicalNetwork(newLn, "UPDATE", "")
	}
}

func (npc *NetworkPolicyController) deleteLogicalNetwork(obj interface{}) {
	l, ok := obj.(*LogicalNetwork)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		l, ok = tombstone.Obj.(*LogicalNetwork)
		if !ok {
			runtime.HandleError(fmt.Errorf("Tombstone contained object that is not a logical network object %#v", obj))
			return
		}
	}

	npc.enqueueLogicalNetwork(l, "DELETE", l.Spec.SubSubnet)
}

func (npc *NetworkPolicyController) addPolicy(obj interface{}) {
	n := obj.(*networkv1.NetworkPolicy)
	npc.enqueueNetworkPolicy(n, "ADD", "")
}

func (npc *NetworkPolicyController) updatePolicy(old, cur interface{}) {
	oldNp := old.(*networkv1.NetworkPolicy)
	newNp := cur.(*networkv1.NetworkPolicy)

	if oldNp.ResourceVersion == newNp.ResourceVersion {
		return
	}

	if oldNp.Annotations != nil || newNp.Annotations != nil {
		if oldNp.Annotations != nil && newNp.Annotations != nil && (oldNp.Annotations[GenieNetworkPolicy] == newNp.Annotations[GenieNetworkPolicy]) {
			return
		} //else

		npc.enqueueNetworkPolicy(newNp, "UPDATE", oldNp.Annotations[GenieNetworkPolicy])
		return
	}

	npc.enqueueNetworkPolicy(newNp, "UPDATE", "")
}

func (npc *NetworkPolicyController) deletePolicy(obj interface{}) {
	n, ok := obj.(*networkv1.NetworkPolicy)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		n, ok = tombstone.Obj.(*networkv1.NetworkPolicy)
		if !ok {
			runtime.HandleError(fmt.Errorf("Tombstone contained object that is not a network policy object %#v", obj))
			return
		}
	}

	if n.Annotations != nil && n.Annotations[GenieNetworkPolicy] != "" {
		npc.enqueueNetworkPolicy(n, "DELETE", n.Annotations[GenieNetworkPolicy])
		return
	}

	npc.enqueueNetworkPolicy(n, "DELETE", "")
}

func (npc *NetworkPolicyController) enqueueLogicalNetwork(lnw *LogicalNetwork, action string, args string) {
	keyaction := map[string]string{"kind": "logicalnetwork", "name": lnw.Name, "namespace": lnw.Namespace, "action": action, "args": args}
	keyactionjson, err := json.Marshal(keyaction)
	if err != nil {
		glog.Warning("Unable to marshal keyaction for logical network: %v", err.Error())
	}

	npc.npcWorkqueue.Add(string(keyactionjson))
}

func (npc *NetworkPolicyController) enqueueNetworkPolicy(np *networkv1.NetworkPolicy, action string, args string) {
	keyaction := map[string]string{"kind": "networkpolicy", "name": np.Name, "namespace": np.Namespace, "action": action, "args": args}
	keyactionjson, err := json.Marshal(keyaction)
	if err != nil {
		glog.Warning("Unable to marshal keyaction for network policy: %v", err.Error())
	}

	npc.npcWorkqueue.Add(string(keyactionjson))
}

func (npc *NetworkPolicyController) Run(n int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer npc.npcWorkqueue.ShutDown()

	glog.Info("Starting network policy controller")

	var err error
	npc.iptable, err = iptables.CreateBaseChain()
	if err != nil {
		return fmt.Errorf("Error creating Genie base npc chain: %v", err)
	}

	glog.Info("Synchronizing informer caches...")
	if ok := cache.WaitForCacheSync(stopCh, npc.networkPoliciesSynced, npc.logicalNwSynced); !ok {
		return fmt.Errorf("Synchronization of informer caches failed.")
	}

	for i := 0; i < n; i++ {
		// Spawn worker threads
		go wait.Until(npc.worker, time.Second, stopCh)
	}

	glog.Info("Started worker threads")
	<-stopCh
	glog.Info("Shutting down worker threads")

	return nil
}

func (npc *NetworkPolicyController) worker() {
	for npc.processNextWorkItemInQueue() {
	}
}

func (npc *NetworkPolicyController) processNextWorkItemInQueue() bool {
	key, quit := npc.npcWorkqueue.Get()
	if quit {
		return false
	}
	defer npc.npcWorkqueue.Done(key)

	err := npc.syncHandler(key.(string))
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error in synchandler: key: %v, error: %v", key, err))
	}
	return true
}

type NetworkPolicy struct {
	NetworkSelector string
	PeerNetworks    string
}

func (npc *NetworkPolicyController) getCidrFromNetwork(name, namespace string) (string, error) {

	lnw, err := npc.logicalNwLister.LogicalNetworks(namespace).Get(name)
	if err != nil {
		return "", err
	}

	return lnw.Spec.SubSubnet, nil

}

type NetworkPolicyInfo struct {
	Name      string
	Namespace string
	Networks  map[string][]string
}

func unmarshalKeyActionJson(key string) (map[string]string, error) {
	var keyaction map[string]string
	err := json.Unmarshal([]byte(key), &keyaction)
	if err != nil {
		return nil, err
	}
	return keyaction, nil
}

func getLogicalNetworksFromAnnotation(annotation string) (map[string][]string, error) {
	networkPolicies := make([]NetworkPolicy, 0)
	policyNetworkMap := make(map[string][]string)

	err := json.Unmarshal([]byte(annotation), &networkPolicies)
	if err != nil {
		return nil, fmt.Errorf("Error while unmarshalling annotation: %v", err)
	}

	for _, policy := range networkPolicies {
		if policy.NetworkSelector != "" {
			policyNetworkMap[policy.NetworkSelector] = append(policyNetworkMap[policy.NetworkSelector], strings.Split(policy.PeerNetworks, ",")...)
		}
	}

	return policyNetworkMap, nil
}

func (npc *NetworkPolicyController) populatePolicyChain(name, namespace, nwPolicyChainName string, networks map[string][]string) error {
	for nwSelector, peerNw := range networks {
		nwSelector = strings.TrimSpace(nwSelector)
		if nwSelector != "" {
			lnChain := iptables.CreateIptableChainName(iptables.GenieNetworkPrefix, nwSelector+namespace)
			if false == npc.iptable.ExistsChain(lnChain) {
				err := npc.handleLogicalNetworkAdd(nwSelector, namespace)
				if err != nil {
					glog.Infof("Skipping handling logical network (%s): %v", nwSelector, err)
					continue
				}
			} else {
				rulespec := []string{"-j", nwPolicyChainName}
				err := npc.iptable.InsertRule(lnChain, 1, rulespec)
				if err != nil {
					glog.Errorf("Error adding rule for policy (%s) in network chain (%s) for logical network (%s): %v", name, lnChain, nwSelector, err)
					continue
				}
			}

			for _, peer := range peerNw {
				l, err := npc.logicalNwLister.LogicalNetworks(namespace).Get(strings.TrimSpace(peer))
				if err != nil {
					continue
				}

				err = npc.insertPeerRule(nwPolicyChainName, l.Spec.SubSubnet)
				if err != nil {
					glog.Errorf("Error adding rule for peer network (%s) in policy chain for policy object (%s:%s): %v", peer, namespace, name)
				}
			}
		}
	}

	return nil
}

func (npc *NetworkPolicyController) handleNetworkPolicyAdd(policyName, policyNamespace string) error {
	networkPolicy, err := npc.networkPoliciesLister.NetworkPolicies(policyNamespace).Get(policyName)
	if err != nil {
		return fmt.Errorf("Failed to get network policy object %s in namespace %s: %v", policyName, policyNamespace, err)
	}

	networks, err := getLogicalNetworksFromAnnotation(networkPolicy.Annotations[GenieNetworkPolicy])
	if err != nil {
		return fmt.Errorf("Error while unmarshalling logical networks info from annotation of policy object %s: %v", networkPolicy.Name, err)
	}

	nwPolicyChainName, err := npc.iptable.AddPolicyChain(policyName, policyNamespace)
	if err != nil {
		return fmt.Errorf("Error adding policy chain for network policy object (%s:%s): %v", policyNamespace, policyName, err)
	}

	if len(networks) > 0 {
		err = npc.populatePolicyChain(policyName, policyNamespace, nwPolicyChainName, networks)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return fmt.Errorf("Error synchronizing network policy: name: %s, namesapce: %s; error: %v", policyName, policyNamespace, err)
	}

	return nil
}

func (npc *NetworkPolicyController) handleNetworkPolicyUpdate(policyName, policyNamespace string) error {
	networkPolicy, err := npc.networkPoliciesLister.NetworkPolicies(policyNamespace).Get(policyName)
	if err != nil {
		return fmt.Errorf("Failed to get network policy object %s in namespace %s: %v", policyName, policyNamespace, err)
	}

	networks, err := getLogicalNetworksFromAnnotation(networkPolicy.Annotations[GenieNetworkPolicy])
	if err != nil {
		return fmt.Errorf("Error while unmarshalling logical networks info from annotation of policy object %s: %v", networkPolicy.Name, err)
	}

	policyChain, err := npc.iptable.AddPolicyChain(policyName, policyNamespace)
	if err != nil {
		return fmt.Errorf("Error reseting policy chain for network policy object (%s:%s): %v", policyNamespace, policyName, err)
	}

	if len(networks) > 0 {
		err = npc.populatePolicyChain(policyName, policyNamespace, policyChain, networks)
		if err != nil {
			return err
		}
	} else {
		rules, err := npc.iptable.List(iptables.FilterTable, iptables.GenieBaseNPCChain)
		if err != nil {
			return fmt.Errorf("Failed to list rules for Genie base chain: %v", err.Error())
		}

		m := make(map[string]bool)
		for _, rule := range rules {
			if strings.Contains(rule, iptables.GenieNetworkPrefix) {
				splitRule := strings.Split(rule, " ")
				var lnChain string
				for _, p := range splitRule {
					if strings.Contains(p, iptables.GenieNetworkPrefix) {
						lnChain = p
					}
				}

				if m[lnChain] == false {
					m[lnChain] = true
					err = npc.iptable.DeleteNetworkChainRule(lnChain, policyChain)
					if err != nil {
						glog.Errorf("Error deleting rule for policy chain (%s) for policy object (%s:%s) form network chain (%s): %v", policyChain, policyNamespace, policyName, lnChain, err)
						continue
					}
				}
			}
		}
	}

	return nil
}

func (npc *NetworkPolicyController) handleNetworkPolicyDelete(policyName, policyNamespace, annotation string) error {
	logicalNetworks, err := getLogicalNetworksFromAnnotation(annotation)
	if err != nil {
		return err
	}

	policyChain := iptables.CreateIptableChainName(iptables.GeniePolicyPrefix, policyName+policyNamespace)

	for lnw := range logicalNetworks {
		nwChain := iptables.CreateIptableChainName(iptables.GenieNetworkPrefix, strings.TrimSpace(lnw)+policyNamespace)
		err = npc.iptable.DeleteNetworkChainRule(nwChain, policyChain)
		if err != nil {
			glog.Errorf("Error while deleting rule for policy (%s:%s) from network chain (%s) for logical network (%s): %v", policyNamespace, policyName, nwChain, lnw, err)
		}
	}

	err = npc.iptable.DeleteIptableChain(iptables.FilterTable, policyChain)
	if err != nil {
		return fmt.Errorf("Error while deleting iptable chain %s for network policy (%s:%s): %v", policyChain, policyNamespace, policyName, err)
	}

	return nil
}

// ListNetworkPolicies lists the network policies which are to be imposed on the given logical network.
// If no logical network name is given then select all the policies which have GenieNetwork Policy annotation
func (npc *NetworkPolicyController) ListNetworkPolicies(lnwname string, namespace string) ([]NetworkPolicyInfo, error) {

	policyInfo := make([]NetworkPolicyInfo, 0)

	networkPolicies, err := npc.networkPoliciesLister.NetworkPolicies(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, policy := range networkPolicies {
		if policy.Annotations == nil || policy.Annotations[GenieNetworkPolicy] == "" {
			continue
		}

		networks, _ := getLogicalNetworksFromAnnotation(policy.Annotations[GenieNetworkPolicy])
		if lnwname != "" {
			if strings.Contains(policy.Annotations[GenieNetworkPolicy], lnwname) {
				if networks[lnwname] != nil {
					policyInfo = append(policyInfo, NetworkPolicyInfo{Name: policy.Name, Namespace: policy.Namespace, Networks: networks})
				} else {
					for _, peers := range networks {
						for _, p := range peers {
							if lnwname == strings.TrimSpace(p) {
								policyInfo = append(policyInfo, NetworkPolicyInfo{Name: policy.Name, Namespace: policy.Namespace, Networks: nil})
							}
						}
					}
				}
			} else {
				continue
			}
		} else {
			policyInfo = append(policyInfo, NetworkPolicyInfo{
				Name:      policy.Name,
				Namespace: policy.Namespace,
				Networks:  networks,
			})
		}
	}

	return policyInfo, nil
}

func (npc *NetworkPolicyController) insertPeerRule(policyChain, subnet string) error {
	rulespec := []string{"-s", subnet, "-j", "ACCEPT"}
	err := npc.iptable.AppendUnique(iptables.FilterTable, policyChain, rulespec...)
	if err != nil {
		return fmt.Errorf("Error adding rule (%v): %v", rulespec, err)
	}
	rulespec = []string{"-d", subnet, "-j", "ACCEPT"}
	err = npc.iptable.AppendUnique(iptables.FilterTable, policyChain, rulespec...)
	if err != nil {
		return fmt.Errorf("Error adding rule (%v): %v", rulespec, err)
	}
	return nil
}

func (npc *NetworkPolicyController) handleLogicalNetworkAdd(name, namespace string) error {
	name = strings.TrimSpace(name)
	logicalNetwork, err := npc.logicalNwLister.LogicalNetworks(namespace).Get(name)
	if err != nil {
		return fmt.Errorf("Error while getting logical network (%s:%s): %v", namespace, name, err)
	}

	policyInfo, err := npc.ListNetworkPolicies(name, namespace)
	if err != nil {
		glog.Errorf("Error listing network policies for logical network (%s): %v", name, err)
	}

	if len(policyInfo) != 0 {
		for _, policy := range policyInfo {
			policyChain := iptables.CreateIptableChainName(iptables.GeniePolicyPrefix, policy.Name+namespace)
			if policy.Networks != nil {
				lnChain, err := npc.iptable.AddNetworkChain(logicalNetwork)

				rulespec := []string{"-j", policyChain}
				err = npc.iptable.InsertRule(lnChain, 1, rulespec)
				if err != nil {
					return fmt.Errorf("Error adding rule for policy (%s) in network chain (%s) for logical network (%s): %v", policy.Name, lnChain, name, err)
				}

				for _, peer := range policy.Networks[name] {
					if peer = strings.TrimSpace(peer); peer != "" {
						peerNw, err := npc.logicalNwLister.LogicalNetworks(namespace).Get(peer)
						if err != nil {
							glog.Errorf("Error while getting logical network (%s:%s) to add rule in policy chain: %v", namespace, peer, err)
							continue
						}
						err = npc.insertPeerRule(policyChain, peerNw.Spec.SubSubnet)
						if err != nil {
							glog.Errorf("Error adding rule for peer network (%s) in policy chain wrt logical network (%s:%s): %v", peer, namespace, name)
						}
					}
				}
			} else {
				err = npc.insertPeerRule(policyChain, logicalNetwork.Spec.SubSubnet)
				if err != nil {
					glog.Errorf("Error adding rule in policy chain (%s) for logical network (%s:%s): %v", policyChain, namespace, name)
				}
			}
		}
	}

	return nil
}

func (npc *NetworkPolicyController) handleLogicalNetworkUpdate(name, namespace string) error {
	logicalNetwork, err := npc.logicalNwLister.LogicalNetworks(namespace).Get(name)
	if err != nil {
		return fmt.Errorf("Error while getting logical network (%s:%s): %v", namespace, name, err)
	}

	lnChain := iptables.CreateIptableChainName(iptables.GenieNetworkPrefix, name+namespace)

	baseChainRules, err := npc.iptable.List(iptables.FilterTable, iptables.GenieBaseNPCChain)
	if err != nil {
		return fmt.Errorf("Error listing Genie base npc chain rules: %v", err.Error())
	}

	var i int
	rulespec := [2][]string{{"-d", logicalNetwork.Spec.SubSubnet, "-j", lnChain}, {"-s", logicalNetwork.Spec.SubSubnet, "-j", lnChain}}
	for pos, rule := range baseChainRules {
		if strings.Contains(rule, lnChain) {
			err = npc.iptable.Delete(iptables.FilterTable, iptables.GenieBaseNPCChain, strconv.Itoa(i))
			if err != nil {
				return fmt.Errorf("Error deleting rule for logical network (%s:%s) from base chain: %v", namespace, name, err)
			}
			err = npc.iptable.Insert(iptables.FilterTable, iptables.GenieBaseNPCChain, pos, rulespec[i]...)
			if err != nil {
				return fmt.Errorf("Error inserting rule for logical network (%s:%s) in base chain: %v", name, namespace, err)
			}
			i++
			if i == 2 {
				break
			}
		}
	}

	return nil
}

func (npc *NetworkPolicyController) handleLogicalNetworkDelete(name, namespace, subnet string) error {
	policyInfo, err := npc.ListNetworkPolicies(name, namespace)
	if err != nil {
		glog.Errorf("Error in ListNetworkPolicies for logical network (%s): %v", name, err)
	}

	if len(policyInfo) != 0 {
		for _, policy := range policyInfo {
			if policy.Networks != nil {
				lnChain := iptables.CreateIptableChainName(iptables.GenieNetworkPrefix, name+namespace)

				err = npc.iptable.DeleteNetworkChain(lnChain)
				if err != nil {
					return fmt.Errorf("Error while deleting iptable chain for logical network (%s:%s): %v", namespace, name, err)
				}
			} else {
				policyChain := iptables.CreateIptableChainName(iptables.GeniePolicyPrefix, policy.Name+policy.Namespace)
				plcRules, err := npc.iptable.List(iptables.FilterTable, policyChain)
				if err != nil {
					glog.Errorf("Failed to list rules for policy chain (%s) for policy (%s:%s): %v", policyChain, policy.Namespace, policy.Name, err)
					continue
				}
				var pos, cnt int
				for _, rule := range plcRules {
					if strings.Contains(rule, subnet) {
						cnt++
						err = npc.iptable.Delete(iptables.FilterTable, policyChain, strconv.Itoa(pos))
						if err != nil {
							glog.Errorf("Failed to remove rule for subnet (%s) of logical network (%s:%s) from policy chain (%s) for policy (%s:%s): %v", subnet, namespace, name, policyChain, namespace, policy.Name, err)
						} else {
							pos--
						}
					}
					if cnt == 2 {
						break
					}
					pos++
				}
			}
		}
	}
	return nil
}

func (npc *NetworkPolicyController) syncHandler(key string) error {

	npc.mutex.Lock()
	defer npc.mutex.Unlock()

	glog.Infof("Starting syncHandler for key: %s", key)
	keyaction, e := unmarshalKeyActionJson(key)
	if e != nil {
		return (fmt.Errorf("Error while unmarshalling action parameters: %v", e))
	}

	var err error
	switch keyaction["kind"] {
	case "networkpolicy":
		switch keyaction["action"] {
		case "ADD":
			err = npc.handleNetworkPolicyAdd(keyaction["name"], keyaction["namespace"])

		case "UPDATE":
			err = npc.handleNetworkPolicyUpdate(keyaction["name"], keyaction["namespace"])

		case "DELETE":
			err = npc.handleNetworkPolicyDelete(keyaction["name"], keyaction["namespace"], keyaction["args"])

		default:

		}

	case "logicalnetwork":
		switch keyaction["action"] {
		case "ADD":
			err = npc.handleLogicalNetworkAdd(keyaction["name"], keyaction["namespace"])

		case "UPDATE":
			err = npc.handleLogicalNetworkUpdate(keyaction["name"], keyaction["namespace"])

		case "DELETE":
			err = npc.handleLogicalNetworkDelete(keyaction["name"], keyaction["namespace"], keyaction["args"])

		default:
		}
	}

	return err
}
