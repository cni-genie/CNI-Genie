package networkcrd

import (
	"github.com/containernetworking/cni/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkAttachmentDefinition describes the network attachment objects
type NetworkAttachmentDefinition struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec specifies an option to choose how the configuration for the
	// CNI plugin, specified by this NetworkAttachmentDefinition object,
	// is to be fetched, i.e. whether from the value defined for the sub-field
	// or to be loaded from the corresponding configuration file present
	// in the network directory, in case the sub-field or this field itself
	// is empty.
	// +optional
	Spec NetworkAttachmentDefinitionSpec `json:"spec"`
}

// NetworkAttachmentDefinitionSpec describes the configuration of network object
type NetworkAttachmentDefinitionSpec struct {
	// Config specifies the configuration in conf or conflist
	// format for the CNI plugin, which will be invoked by this
	// NetworkAttachmentDefinition object
	// +optional
	Config string `json:"config,omitempty"`
}

// NetworkSelectionElement describes the properties of a single object specified
// in network attachment selection annotation
type NetworkSelectionElement struct {
	// Name specifies the name of a NetworkAttachmentDefinition object to be selected
	Name string `json:"name"`
	// Namespace specifies the namespace of the NetworkAttachmentDefinition object named by the "name" key.
	// Defaults to pod's namespace if this key is not specified
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// IPs specifies an array of user requested IP addresses to be assigned to the pod
	// by the plugin handling this network attachment
	// +optional
	IPs []string `json:"ips,omitempty"`
	// Mac specifies an user requested mac address to be assigned to the pod
	// by the plugin handling this network attachment
	// +optional
	Mac string `json:"mac,omitempty"`
	// Interface specifies the user requested name to be used for the interface assigned
	// to the container by this network attachment
	// +optional
	Interface string `json:"interface,omitempty"`
}

// NetworkStatus describes the status to be updated in pod
type NetworkStatus struct {
	Name      string    `json:"name"`
	Interface string    `json:"interface,omitempty"`
	IPs       []string  `json:"ips,omitempty"`
	Mac       string    `json:"mac,omitempty"`
	Default   bool      `json:"default,omitempty"`
	DNS       types.DNS `json:"dns,omitempty"`
}
