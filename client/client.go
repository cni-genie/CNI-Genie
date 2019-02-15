package client

import (
	"fmt"
	"github.com/cni-genie/CNI-Genie/utils"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strings"
)

type ClientInterface interface {
	GetPod(name, namespace string) (*v1.Pod, error)
	PatchPod(name, namespace string, pt types.PatchType, data []byte) (*v1.Pod, error)
	GetRaw(path string) ([]byte, error)
}

type KubeClient struct {
	kubernetes.Interface
}

func (kc *KubeClient) GetPod(name, namespace string) (*v1.Pod, error) {
	pod, err := kc.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func (kc *KubeClient) PatchPod(name, namespace string, pt types.PatchType, data []byte) (*v1.Pod, error) {
	pod, err := kc.CoreV1().Pods(namespace).Patch(name, pt, data)
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func (kc *KubeClient) GetRaw(path string) ([]byte, error) {
	obj, err := kc.ExtensionsV1beta1().RESTClient().Get().AbsPath(path).DoRaw()
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// GetKubeClient creates a kubeclient from genie-kubeconfig file,
// default location is /etc/cni/net.d.
func BuildKubeClientFromConfig(conf *utils.GenieConf) (*KubeClient, error) {
	config, err := buildKubeConfig(conf)
	if err != nil {
		return nil, err
	}
	// Create the clientset
	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &KubeClient{kc}, nil
}

func buildKubeConfig(conf *utils.GenieConf) (*restclient.Config, error) {
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

	fmt.Fprintf(os.Stderr, "CNI Genie Kubernetes config %v\n", config)

	return config, nil
}
