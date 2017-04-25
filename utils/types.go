package utils

import (
	"net"
	"github.com/containernetworking/cni/pkg/types"
	v1 "github.com/projectcalico/cni-plugin/utils"
)

// NetConf stores the common network config for Calico CNI plugin
type NetConf struct {
	CNIVersion string `json:"cniVersion"`
	Name string `json:"name"`
	Type string `json:"type"`
	IPAM struct {
		     Name       string
		     Type       string   `json:"type"`
		     Subnet     string   `json:"subnet"`
		     AssignIpv4 *string  `json:"assign_ipv4"`
		     AssignIpv6 *string  `json:"assign_ipv6"`
		     IPv4Pools  []string `json:"ipv4_pools,omitempty"`
		     IPv6Pools  []string `json:"ipv6_pools,omitempty"`
	     } `json:"ipam,omitempty"`
	MTU            int        `json:"mtu"`
	Hostname       string     `json:"hostname"`
	DatastoreType  string     `json:"datastore_type"`
	EtcdAuthority  string     `json:"etcd_authority"`
	EtcdEndpoints  string     `json:"etcd_endpoints"`
	LogLevel       string     `json:"log_level"`
	Policy         v1.Policy     `json:"policy"`
	Kubernetes     v1.Kubernetes `json:"kubernetes"`
	Args           v1.Args       `json:"args"`
	EtcdScheme     string     `json:"etcd_scheme"`
	EtcdKeyFile    string     `json:"etcd_key_file"`
	EtcdCertFile   string     `json:"etcd_cert_file"`
	EtcdCaCertFile string     `json:"etcd_ca_cert_file"`
	Delegate       v1.Delegate   `json:"delegate"`
}

// K8sArgs is the valid CNI_ARGS used for Kubernetes
type K8sArgs struct {
	types.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               types.UnmarshallableString
	K8S_POD_NAMESPACE          types.UnmarshallableString
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString
}