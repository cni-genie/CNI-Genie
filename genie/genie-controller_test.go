package genie

import (
	"fmt"
	"github.com/cni-genie/CNI-Genie/client"
	it "github.com/cni-genie/CNI-Genie/interfaces"
	"github.com/cni-genie/CNI-Genie/networkcrd"
	"github.com/cni-genie/CNI-Genie/utils"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"strings"
	"testing"
)

const (
	CniAnnotation          = "cni"
	NetworkAnnotation      = "networks"
	NetAttachDefAnnotation = "k8s.v1.cni.cncf.io/networks"
)

var defaultGenieConf *utils.GenieConf = &utils.GenieConf{
	NetConf: types.NetConf{
		Name: "k8s-pod-network",
		Type: "genie",
	},
}

func newController(plugins []string, obj ...runtime.Object) *GenieController {
	gc := &GenieController{
		Invoke: &it.FakeInvoke{},
		Cfg: &it.CNIConfig{
			CNI:    &it.FakeCni{InstalledPlugins: plugins},
			RW:     &it.FakeIo{Files: plugins},
			NetDir: DefaultNetDir,
			BinDir: DefaultPluginDir,
		},
	}

	gc.Kc = &client.KubeClient{Interface: fake.NewSimpleClientset(obj...)}
	it.SetIp(plugins)

	return gc
}

func newPod(name, namespace string, annot map[string]string) *v1.Pod {
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annot,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Image: "abc",
				},
			},
		},
	}

	return pod
}

func getCniArgs(podName, podNamespace string) *utils.CNIArgs {
	cmdArgs := &skel.CmdArgs{
		Args: "K8S_POD_NAME=" + podName + ";K8S_POD_NAMESPACE=" + podNamespace,
	}

	return PopulateCNIArgs(cmdArgs)
}

func compareErrors(expected, actual error) bool {
	if expected == nil && actual == nil {
		return true
	} else if expected != nil && actual != nil {
		return strings.Contains(actual.Error(), expected.Error())
	}

	return false
}

func TestAddNetwork(t *testing.T) {

	tests := []struct {
		pod               *v1.Pod
		logNetworks       []*utils.LogicalNetwork
		phyNetworks       []*utils.PhysicalNetwork
		netAttachDefs     []*networkcrd.NetworkAttachmentDefinition
		pluginsInstalled  []string
		genieConf         *utils.GenieConf
		expectedStatusLen int
		expectedErr       error
	}{
		{
			pod:               newPod("testpod", "default", map[string]string{CniAnnotation: "flannel, weave, abc, flannel"}),
			logNetworks:       nil,
			phyNetworks:       nil,
			netAttachDefs:     nil,
			pluginsInstalled:  []string{"flannel", "weave", "calico", "abc"},
			genieConf:         defaultGenieConf,
			expectedStatusLen: 2,
			expectedErr:       nil,
		},
	}

	for i := range tests {
		objs := make([]runtime.Object, 0)
		objs = append(objs, tests[i].pod)
		gc := newController(tests[i].pluginsInstalled, objs...)

		cniArgs := getCniArgs(tests[i].pod.Name, tests[i].pod.Namespace)
		res, err := gc.AddPodNetwork(cniArgs, tests[i].genieConf)
		if false == compareErrors(tests[i].expectedErr, err) {
			t.Errorf("Expected error: %v; got error: %v", tests[i].expectedErr, err)
		}
		fmt.Println("result: ", res)
	}
}
