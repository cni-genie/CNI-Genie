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

	clientset "github.com/cni-genie/CNI-Genie/controllers/logicalnetwork-pkg/client/clientset/versioned"
	networkscheme "github.com/cni-genie/CNI-Genie/controllers/logicalnetwork-pkg/client/clientset/versioned/scheme"
	informers "github.com/cni-genie/CNI-Genie/controllers/logicalnetwork-pkg/client/informers/externalversions"
	listers "github.com/cni-genie/CNI-Genie/controllers/logicalnetwork-pkg/client/listers/network/v1"

	"encoding/json"
	iptables "github.com/cni-genie/CNI-Genie/controllers/network-policy-controller/iptables"
	. "github.com/cni-genie/CNI-Genie/utils"
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

type NetworkPolicy struct {
	NetworkSelector string `json:"networkSelector,omitempty"`
	PeerNetworks    string `json:"peerNetworks,omitempty"`
}

type AsSelector struct {
	name      string
	namespace string
}

type AsPeer struct {
	name      string
	namespace string
	selector  string
}

type NetworkPolicyInfo struct {
	AsSelector bool
	AsPeer     []string
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

		npc.enqueueNetworkPolicy(newNp, "UPDATE", GenieNetworkPolicy)
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
		npc.enqueueNetworkPolicy(n, "DELETE", GenieNetworkPolicy)
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

func unmarshalKeyActionJson(key string) (map[string]string, error) {
	var keyaction map[string]string
	err := json.Unmarshal([]byte(key), &keyaction)
	if err != nil {
		return nil, err
	}
	return keyaction, nil
}

func parsePeers(peers string) string {
	peer := strings.Split(peers, ",")
	var ret string
	for _, p := range peer {
		ret += strings.TrimSpace(p)
	}

	return ret
}

func getLogicalNetworksFromAnnotation(annotation string) (map[string][]string, error) {
	networkPolicies := make([]NetworkPolicy, 0)
	policyNetworkMap := make(map[string][]string)

	glog.V(4).Infof("Unmarshalling annotation: %s", annotation)
	err := json.Unmarshal([]byte(annotation), &networkPolicies)
	if err != nil {
		return nil, fmt.Errorf("Error while unmarshalling annotation: %v", err)
	}

	for _, policy := range networkPolicies {
		nwSelector := strings.TrimSpace(policy.NetworkSelector)
		if nwSelector != "" {
			policyNetworkMap[nwSelector] = append(policyNetworkMap[policy.NetworkSelector], strings.Split(parsePeers(policy.PeerNetworks), ",")...)
		}
	}
	glog.V(4).Infof("Unmarshalled logical network map from annotation: %+v", policyNetworkMap)
	return policyNetworkMap, nil
}

func getNetworkInfoFromAnnotation(annotation, nwName string) (NetworkPolicyInfo, error) {
	networkPolicies := make([]NetworkPolicy, 0)
	glog.V(4).Infof("Unmarshalling annotation: %s", annotation)
	err := json.Unmarshal([]byte(annotation), &networkPolicies)
	if err != nil {
		return NetworkPolicyInfo{}, fmt.Errorf("Error while unmarshalling annotation: %v", err)
	}

	networkInfo := NetworkPolicyInfo{}
	for _, policyRule := range networkPolicies {
		peers := parsePeers(policyRule.PeerNetworks)
		if nwName == policyRule.NetworkSelector && false == networkInfo.AsSelector {
			networkInfo.AsSelector = true
		} else if strings.Contains(","+peers+",", ","+nwName+",") {
			networkInfo.AsPeer = append(networkInfo.AsPeer, strings.TrimSpace(policyRule.NetworkSelector))

		}
	}

	return networkInfo, nil
}

func (npc *NetworkPolicyController) populatePolicyChain(name, namespace string, networks map[string][]string) ([]string, error) {
	policyChainsAdded := make([]string, 0)
	for nwSelector, peerNw := range networks {
		nwSelector = strings.TrimSpace(nwSelector)
		if nwSelector != "" {
			selectorNw, err := npc.logicalNwLister.LogicalNetworks(namespace).Get(nwSelector)
			if err != nil {
				glog.Errorf("Error getting selector logical network (%s:%s): %v", namespace, nwSelector, err)
				continue
			}
			nwPolicyChainName, err := npc.iptable.AddPolicyChain(name, namespace, nwSelector)
			if err != nil {
				return nil, fmt.Errorf("Error adding policy chain for network policy object (%s:%s): %v", name, namespace, err)
			}
			policyChainsAdded = append(policyChainsAdded, nwPolicyChainName)
			for _, peer := range peerNw {
				if peer = strings.TrimSpace(peer); peer == "" {
					continue
				}
				l, err := npc.logicalNwLister.LogicalNetworks(namespace).Get(peer)
				if err != nil {
					glog.Errorf("Error getting peer logical network (%s:%s): %v", namespace, peer, err)
					continue
				}

				err = npc.insertPeerRule(nwPolicyChainName, selectorNw.Spec.SubSubnet, l.Spec.SubSubnet)
				if err != nil {
					glog.Errorf("Error adding rule (subnet: %s) for peer network (%s:%s) in policy chain for policy object (%s:%s): %v", l.Spec.SubSubnet, namespace, peer, namespace, name, err)
				}
			}
			glog.V(4).Infof("Finished preparing policy chain (%s) for policy object (%s:%s)", nwPolicyChainName, namespace, name)

			lnChain := iptables.CreateIptableChainName(iptables.GenieNetworkPrefix, nwSelector+namespace)
			if false == npc.iptable.ExistsChain(lnChain) {
				glog.V(6).Infof("Network chain %s does not exist. So trying to add it", lnChain)
				err := npc.handleLogicalNetworkAdd(nwSelector, namespace)
				if err != nil {
					glog.Errorf("Skipping handling logical network (%s:%s) as part of handling policy object (%s:%s): %v", namespace, nwSelector, namespace, name, err)
				}
			} else {
				glog.V(6).Infof("Network chain %s exists. Adding rule to it.", lnChain)
				rulespec := []string{"-j", nwPolicyChainName}
				err := npc.iptable.InsertRule(lnChain, 1, rulespec)
				if err != nil {
					glog.Errorf("Error adding rule (policy chain: %s) for policy (%s:%s) in network chain (%s) for logical network (%s:%s): %v", nwPolicyChainName, namespace, name, lnChain, namespace, nwSelector, err)
				}
			}
		}
	}

	return policyChainsAdded, nil
}

func (npc *NetworkPolicyController) removePolicyChainEntries(policyChains []string) error {
	baseRules, err := npc.iptable.List(iptables.FilterTable, iptables.GenieBaseNPCChain)
	if err != nil {
		return fmt.Errorf("Failed to list rules for Genie base chain: %v", err)
	}
	glog.V(4).Infof("Entries for the policy chains to be removed are: %v", policyChains)
	nwChains := make(map[string]bool)
	policyChainsToDelete := make(map[string]bool)
	for _, rule := range baseRules {
		if strings.HasPrefix(rule, "-A") && strings.Contains(rule, iptables.GenieNetworkPrefix) {
			lnChain := rule[strings.LastIndex(rule, " ")+1:]
			if nwChains[lnChain] == false {
				nwChains[lnChain] = true
				glog.V(4).Infof("Removing policy chain entries from network chain (%s)", lnChain)
				rulesDeleted, err := npc.iptable.DeleteNetworkChainRule(lnChain, policyChains)
				if err != nil {
					glog.Errorf("Error removing rules from network chain (%s): %v", lnChain, err)
				}
				for _, r := range rulesDeleted {
					policyChainsToDelete[r] = true
				}
			}
		}
	}

	glog.V(4).Infof("Policy chains to be deleted from iptable: %v", policyChainsToDelete)
	for policyChain := range policyChainsToDelete {
		err := npc.iptable.DeleteIptableChain(iptables.FilterTable, policyChain)
		if err != nil {
			glog.Errorf("Error deleting policy chain (%s) from iptable: %v", policyChain, err)
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

	if len(networks) > 0 {
		_, err = npc.populatePolicyChain(policyName, policyNamespace, networks)
		if err != nil {
			return fmt.Errorf("Error adding policy chain entries for policy object (%s:%s): %v", policyNamespace, policyName, err)
		}
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

	iptableChains, err := npc.iptable.ListChains(iptables.FilterTable)
	if err != nil {
		glog.Errorf("Error while listing chains in filter table: %v", err)
	}

	var policyChains []string
	if len(networks) > 0 {
		policyChains, err = npc.populatePolicyChain(policyName, policyNamespace, networks)
		if err != nil {
			return err
		}
	}
	glog.V(6).Infof("Policy chains added/updated: %v", policyChains)
	plcChainFmt := iptables.CreatePolicyChainName(policyName, policyNamespace, "")
	plcChainFmt = plcChainFmt[:strings.LastIndex(plcChainFmt, "-")+1]
	newChains := "," + strings.Join(policyChains, ",") + ","

	policyChainsToRemove := make([]string, 0)
	for _, chain := range iptableChains {
		if strings.Contains(chain, plcChainFmt) && !strings.Contains(newChains, ","+chain+",") {
			policyChainsToRemove = append(policyChainsToRemove, chain)

		}
	}

	glog.V(6).Infof("Removing entries of policy chains: %v", policyChainsToRemove)
	err = npc.removePolicyChainEntries(policyChainsToRemove)
	return nil
}

func (npc *NetworkPolicyController) handleNetworkPolicyDelete(policyName, policyNamespace string) error {
	plcChainFmt := iptables.CreatePolicyChainName(policyName, policyNamespace, "")
	plcChainFmt = plcChainFmt[:strings.LastIndex(plcChainFmt, "-")+1]

	glog.V(4).Infof("Deleting policy chain (%s) entries for policy object (%s:%s) as part of processing policy object deletion", plcChainFmt, policyNamespace, policyName)
	err := npc.removePolicyChainEntries([]string{plcChainFmt})
	if err != nil {
		return fmt.Errorf("Error while deleting policy chain and its entries for policy object (%s:%s): %v", policyNamespace, policyName, err)
	}

	return nil
}

// ListNetworkPolicies lists the network policies which are to be imposed on the given logical network.
// If no logical network name is given then select all the policies which have GenieNetwork Policy annotation
func (npc *NetworkPolicyController) ListNetworkPolicies(lnwname string, namespace string) ([]AsSelector, []AsPeer, error) {
	networkPolicies, err := npc.networkPoliciesLister.NetworkPolicies(namespace).List(labels.Everything())
	if err != nil {
		return nil, nil, err
	}

	asSelector := make([]AsSelector, 0)
	asPeer := make([]AsPeer, 0)
	for _, policy := range networkPolicies {
		if policy.Annotations == nil || policy.Annotations[GenieNetworkPolicy] == "" {
			continue
		}

		networkInfo, err := getNetworkInfoFromAnnotation(policy.Annotations[GenieNetworkPolicy], lnwname)
		if err != nil {
			glog.Errorf("Error parsing logical network info from annotation: %v", err)
			continue
		}

		if networkInfo.AsSelector == true {
			asSelector = append(asSelector, AsSelector{name: policy.Name, namespace: policy.Namespace})
		}
		if len(networkInfo.AsPeer) > 0 {
			for _, selector := range networkInfo.AsPeer {
				asPeer = append(asPeer, AsPeer{name: policy.Name, namespace: policy.Namespace, selector: selector})
			}
		}
	}

	return asSelector, asPeer, nil
}

func (npc *NetworkPolicyController) insertPeerRule(policyChain, selectorSubnet, peerSubnet string) error {
	rulespec := []string{"-s", selectorSubnet, "-d", peerSubnet, "-j", "ACCEPT"}
	err := npc.iptable.AppendUnique(iptables.FilterTable, policyChain, rulespec...)
	if err != nil {
		return fmt.Errorf("Error adding rule (%v): %v", rulespec, err)
	}
	rulespec = []string{"-s", peerSubnet, "-d", selectorSubnet, "-j", "ACCEPT"}
	err = npc.iptable.AppendUnique(iptables.FilterTable, policyChain, rulespec...)
	if err != nil {
		return fmt.Errorf("Error adding rule (%v): %v", rulespec, err)
	}
	return nil
}

func (npc *NetworkPolicyController) deletePeerRule(policyChain, subnet string) error {
	plcRules, err := npc.iptable.List(iptables.FilterTable, policyChain)
	if err != nil {
		return fmt.Errorf("Failed to list rules for policy chain (%s): %v", policyChain, err)
	}
	var pos, cnt int
	for _, rule := range plcRules {
		if strings.Contains(rule, subnet) {
			cnt++
			err = npc.iptable.Delete(iptables.FilterTable, policyChain, strconv.Itoa(pos))
			if err != nil {
				return fmt.Errorf("Failed to remove rule (%s) from policy chain (%s): %v", rule, policyChain, err)
			} else {
				pos--
			}
		}
		if cnt == 2 {
			break
		}
		pos++
	}
	return nil
}

func (npc *NetworkPolicyController) handleLogicalNetworkAdd(name, namespace string) error {
	name = strings.TrimSpace(name)
	logicalNetwork, err := npc.logicalNwLister.LogicalNetworks(namespace).Get(name)
	if err != nil {
		return fmt.Errorf("Error while getting logical network (%s:%s): %v", namespace, name, err)
	}

	asSelector, asPeer, err := npc.ListNetworkPolicies(name, namespace)
	if err != nil {
		glog.Errorf("Error listing network policies for logical network (%s): %v", name, err)
	}
	glog.V(6).Infof("Logical network (%s:%s) is a selector network for these policy objects: %v", namespace, name, asSelector)
	glog.V(6).Infof("Logical network (%s:%s) is a peer network for these slector-policy object combinations: %v", namespace, name, asPeer)

	if len(asSelector) > 0 {
		lnChain, err := npc.iptable.AddNetworkChain(logicalNetwork)
		glog.V(6).Infof("Added network chain %s for selector network %s", lnChain, name)
		if err != nil {
			return fmt.Errorf("Error adding network chain for logical network (%s:%s): %v", logicalNetwork.Namespace, logicalNetwork.Name, err)
		}
		for _, policy := range asSelector {
			policyChain := iptables.CreatePolicyChainName(policy.name, policy.namespace, name)
			if npc.iptable.ExistsChain(policyChain) {
				glog.V(6).Infof("Adding policy rule %s to network chain %s for selector network %s", policyChain, lnChain, name)
				rulespec := []string{"-j", policyChain}
				err = npc.iptable.InsertRule(lnChain, 1, rulespec)
				if err != nil {
					return fmt.Errorf("Error adding rule (%s) for policy object (%s:%s) in network chain (%s) for logical network (%s:%s): %v", policyChain, policy.namespace, policy.name, lnChain, namespace, name, err)
				}
			}
		}
	}

	if len(asPeer) > 0 {
		for _, policy := range asPeer {
			selectorNw, err := npc.logicalNwLister.LogicalNetworks(namespace).Get(policy.selector)
			if err != nil {
				glog.Errorf("Error while getting selector logical network (%s:%s) before adding peer rule: %v", namespace, policy.selector, err)
				continue
			}
			policyChain := iptables.CreatePolicyChainName(policy.name, policy.namespace, policy.selector)
			if npc.iptable.ExistsChain(policyChain) {
				err = npc.insertPeerRule(policyChain, selectorNw.Spec.SubSubnet, logicalNetwork.Spec.SubSubnet)
				if err != nil {
					glog.Errorf("Error adding rule in policy chain (%s) for peer logical network (%s:%s): %v", policyChain, namespace, name, err)
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
	asSelector, asPeer, err := npc.ListNetworkPolicies(name, namespace)
	if err != nil {
		glog.Errorf("Error in ListNetworkPolicies for logical network (%s): %v", name, err)
	}

	if len(asSelector) > 0 {
		lnChain := iptables.CreateIptableChainName(iptables.GenieNetworkPrefix, name+namespace)
		err = npc.iptable.DeleteNetworkChain(lnChain)
		if err != nil {
			return fmt.Errorf("Error while deleting iptable chain for logical network (%s:%s): %v", namespace, name, err)
		}

		for _, policy := range asSelector {
			policyChain := iptables.CreatePolicyChainName(policy.name, policy.namespace, name)
			err = npc.iptable.DeleteIptableChain(iptables.FilterTable, policyChain)
			if err != nil {
				glog.Errorf("Error deleting policy chain (%s) for policy (%s:%s) as part of selector logical network (%s:%s) deletion: %v", policyChain, policy.namespace, policy.name, namespace, name, err)
			}
		}
	}

	if len(asPeer) > 0 {
		for _, policy := range asPeer {
			policyChain := iptables.CreatePolicyChainName(policy.name, policy.namespace, policy.selector)
			err = npc.deletePeerRule(policyChain, subnet)
			if err != nil {
				glog.Errorf("Error deleting rule (subnet: %s) for peer logical network (%s:%s) form policy chain (%s) for policy (%s:%s): %v", subnet, namespace, name, policyChain, policy.namespace, policy.name, err)
				continue
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
			err = npc.handleNetworkPolicyDelete(keyaction["name"], keyaction["namespace"])

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
