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
	scheme "github.com/cni-genie/CNI-Genie/controllers/logicalnetwork-pkg/client/clientset/versioned/scheme"
	v1 "github.com/cni-genie/CNI-Genie/utils"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// LogicalNetworksGetter has a method to return a LogicalNetworkInterface.
// A group's client should implement this interface.
type LogicalNetworksGetter interface {
	LogicalNetworks(namespace string) LogicalNetworkInterface
}

// LogicalNetworkInterface has methods to work with LogicalNetwork resources.
type LogicalNetworkInterface interface {
	Create(*v1.LogicalNetwork) (*v1.LogicalNetwork, error)
	Update(*v1.LogicalNetwork) (*v1.LogicalNetwork, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.LogicalNetwork, error)
	List(opts meta_v1.ListOptions) (*v1.LogicalNetworkList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.LogicalNetwork, err error)
	LogicalNetworkExpansion
}

// logicalNetworks implements LogicalNetworkInterface
type logicalNetworks struct {
	client rest.Interface
	ns     string
}

// newLogicalNetworks returns a LogicalNetworks
func newLogicalNetworks(c *AlphaV1Client, namespace string) *logicalNetworks {
	return &logicalNetworks{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the logicalNetwork, and returns the corresponding logicalNetwork object, and an error if there is any.
func (c *logicalNetworks) Get(name string, options meta_v1.GetOptions) (result *v1.LogicalNetwork, err error) {
	result = &v1.LogicalNetwork{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("logicalnetworks").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of LogicalNetworks that match those selectors.
func (c *logicalNetworks) List(opts meta_v1.ListOptions) (result *v1.LogicalNetworkList, err error) {
	result = &v1.LogicalNetworkList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("logicalnetworks").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested logicalNetworks.
func (c *logicalNetworks) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("logicalnetworks").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a logicalNetwork and creates it.  Returns the server's representation of the logicalNetwork, and an error, if there is any.
func (c *logicalNetworks) Create(logicalNetwork *v1.LogicalNetwork) (result *v1.LogicalNetwork, err error) {
	result = &v1.LogicalNetwork{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("logicalnetworks").
		Body(logicalNetwork).
		Do().
		Into(result)
	return
}

// Update takes the representation of a logicalNetwork and updates it. Returns the server's representation of the logicalNetwork, and an error, if there is any.
func (c *logicalNetworks) Update(logicalNetwork *v1.LogicalNetwork) (result *v1.LogicalNetwork, err error) {
	result = &v1.LogicalNetwork{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("logicalnetworks").
		Name(logicalNetwork.Name).
		Body(logicalNetwork).
		Do().
		Into(result)
	return
}

// Delete takes name of the logicalNetwork and deletes it. Returns an error if one occurs.
func (c *logicalNetworks) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("logicalnetworks").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *logicalNetworks) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("logicalnetworks").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched logicalNetwork.
func (c *logicalNetworks) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.LogicalNetwork, err error) {
	result = &v1.LogicalNetwork{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("logicalnetworks").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
