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

package v1

import (
	time "time"

	versioned "github.com/cni-genie/CNI-Genie/controllers/logicalnetwork-pkg/client/clientset/versioned"
	internalinterfaces "github.com/cni-genie/CNI-Genie/controllers/logicalnetwork-pkg/client/informers/externalversions/internalinterfaces"
	v1 "github.com/cni-genie/CNI-Genie/controllers/logicalnetwork-pkg/client/listers/network/v1"
	network_v1 "github.com/cni-genie/CNI-Genie/utils"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// LogicalNetworkInformer provides access to a shared informer and lister for
// LogicalNetworks.
type LogicalNetworkInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.LogicalNetworkLister
}

type logicalNetworkInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewLogicalNetworkInformer constructs a new informer for LogicalNetwork type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewLogicalNetworkInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredLogicalNetworkInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredLogicalNetworkInformer constructs a new informer for LogicalNetwork type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredLogicalNetworkInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AlphaV1().LogicalNetworks(namespace).List(options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AlphaV1().LogicalNetworks(namespace).Watch(options)
			},
		},
		&network_v1.LogicalNetwork{},
		resyncPeriod,
		indexers,
	)
}

func (f *logicalNetworkInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredLogicalNetworkInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *logicalNetworkInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&network_v1.LogicalNetwork{}, f.defaultInformer)
}

func (f *logicalNetworkInformer) Lister() v1.LogicalNetworkLister {
	return v1.NewLogicalNetworkLister(f.Informer().GetIndexer())
}
