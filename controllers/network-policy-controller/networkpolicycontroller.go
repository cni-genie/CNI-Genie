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
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	networklisters "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	networkv1 "k8s.io/api/networking/v1"

	clientset "github.com/Huawei-PaaS/CNI-Genie/controllers/logicalnetwork-pkg/client/clientset/versioned"
	networkscheme "github.com/Huawei-PaaS/CNI-Genie/controllers/logicalnetwork-pkg/client/clientset/versioned/scheme"
	informers "github.com/Huawei-PaaS/CNI-Genie/controllers/logicalnetwork-pkg/client/informers/externalversions"
	listers "github.com/Huawei-PaaS/CNI-Genie/controllers/logicalnetwork-pkg/client/listers/network/v1"

	. "github.com/Huawei-PaaS/CNI-Genie/utils"
	"time"
	"encoding/json"
)

const (
	ControllerAgentName = "network-policy-controller"
	GenieNetworkPolicy = "genieNetworkPolicy"
	GeniePolicyPrefix = "GeniePolicy"
	GenieNetworkPrefix = "GenieNetwork"
)

type NetworkPolicyController struct {
	kubeclientset kubernetes.Interface
	extclientset clientset.Interface

	networkPoliciesLister networklisters.NetworkPolicyLister
	networkPoliciesSynced cache.InformerSynced
	logicalNwLister        listers.LogicalNetworkLister
	logicalNwSynced        cache.InformerSynced

	npcWorkqueue workqueue.RateLimitingInterface
	recorder record.EventRecorder

	mutex sync.Mutex
}

type NetworkPolicyInfo struct {
	Name string
	Namespace string
	Networks map[string][]string
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
		kubeclientset:     kubeclientset,
		extclientset:   extclientset,
		networkPoliciesLister: networkPolicyInformer.Lister(),
		networkPoliciesSynced: networkPolicyInformer.Informer().HasSynced,
		logicalNwLister:        logicalNwInformer.Lister(),
		logicalNwSynced:        logicalNwInformer.Informer().HasSynced,
		npcWorkqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "npc"),
		recorder:          recorder,
	}

	logicalNwInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: npcController.addLogicalNetwork,
		UpdateFunc: npcController.updateLogicalNetwork,
		DeleteFunc: npcController.deleteLogicalNetwork,
	})

	networkPolicyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: npcController.addPolicy,
		UpdateFunc:npcController.updatePolicy,
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

	if n.Annotations != nil && n.Annotations[GeniePolicyPrefix] != "" {
		npc.enqueueNetworkPolicy(n, "DELETE", n.Annotations[GenieNetworkPolicy])
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
	runtime.HandleError(fmt.Errorf("Error in synchandler: key: %v, error: %v", key, err))

	return true
}

func (npc *NetworkPolicyController) syncHandler(keyaction interface{}) error {

	return nil
}
