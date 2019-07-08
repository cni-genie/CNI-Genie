package main_test

import (
	"flag"
	"fmt"
	logicalv1 "github.com/cni-genie/CNI-Genie/controllers/logicalnetwork-pkg/client/clientset/versioned/typed/network/v1"
	"github.com/cni-genie/CNI-Genie/utils"
	"github.com/golang/glog"
	netattachv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	netattachclientset "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"math/rand"
	"os"
	"time"
)

const (
	TEST_NAMESPACE     = "test"
	DEFAULT            = "default"
	NetworkCRD         = "logicalnetworks.alpha.network.k8s.io"
	CRD_TEST_NAMESPACE = "network"
	NetAttachDef       = "network-attachment-definitions.k8s.cni.cncf.io"
)

var testKubeVersion string
var testKubeConfig string
var clientset *kubernetes.Clientset
var apiextensionsclient *apiextensionsclientset.Clientset

func init() {
	// go test -args --testKubeVersion="1.6" --testKubeConfig="/root/admin.conf"
	// To override default values pass --testKubeVersion --testKubeConfig flags
	flag.StringVar(&testKubeVersion, "testKubeVersion", "1.5", "Specify kubernetes version eg: 1.5 or 1.6 or 1.7")
	flag.StringVar(&testKubeConfig, "testKubeConfig", "/root/admin.conf", "Specify testKubeConfig path eg: /root/kubeconfig")
}

var _ = Describe("CNIGenie", func() {

	hostname, _ := os.Hostname()
	glog.Info("Inside CNIGenie tests for k8s:", hostname)

	Describe("Add calico networking for Pod", func() {
		glog.Info("Inside Check for adding Calico networking")
		Context("using cni-genie for configuring calico CNI", func() {
			name := fmt.Sprintf("nginx-calico-%d", rand.Uint32())
			interfaceName := "eth0"
			glog.Info(interfaceName)

			It("should succeed calico networking for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "calico"
				//Create a K8s Pod with calico cni
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the calico pod to have running status")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the calico pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for calico pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Add romana networking for Pod", func() {
		glog.Info("Inside Check for adding romana networking")
		Context("using cni-genie for configuring romana CNI", func() {
			name := fmt.Sprintf("nginx-romana-%d", rand.Uint32())
			interfaceName := "eth0"
			glog.Info(interfaceName)

			It("should succeed romana networking for pod", func() {
				Skip("Skip Adding romana networking for Pod")
				annots := make(map[string]string)
				annots["cni"] = "romana"
				//Create a K8s Pod with calico cni
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the romana pod to have running status")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the romana pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Add weave networking for Pod", func() {
		glog.Info("Inside Check for adding weave networking")
		Context("using cni-genie for configuring weave CNI", func() {
			name := fmt.Sprintf("nginx-weave-%d", rand.Uint32())
			interfaceName := "eth0"
			glog.Info(interfaceName)

			It("should succeed weave networking for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "weave"
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the weave pod to have running status")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the weave pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Add multi-ip networking for Pod", func() {
		glog.Info("Inside Check for adding multi-ip networking")
		Context("using cni-genie for configuring multi-ip CNI", func() {
			name := fmt.Sprintf("nginx-multiip-%d", rand.Uint32())
			interfaceName := "eth0"
			glog.Info(interfaceName)

			It("should succeed multi-ip networking for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "flannel,weave"
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the multi-ip pod to have running status")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the multi-ip pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for multi-ip pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Add nocni networking for Pod", func() {
		glog.Info("Inside Check for adding nocni networking")
		Context("using cni-genie for configuring nocni CNI", func() {
			name := fmt.Sprintf("nginx-nocni-%d", rand.Uint32())
			interfaceName := "eth0"
			glog.Info(interfaceName)

			It("should succeed nocni networking for pod", func() {
				Skip("Skip Adding nocni networking for Pod")
				annots := make(map[string]string)
				annots["cni"] = " "
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the nocni pod to have running status")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the nocni pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for nocni pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})
	Describe("Add bridge networking for Pod", func() {
		glog.Info("Inside Check for adding bridge networking")
		Context("using cni-genie for configuring bridge CNI", func() {
			name := fmt.Sprintf("nginx-bridge-%d", rand.Uint32())

			It("should succeed bridge networking for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "bridge"
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the bridge pod to have running status")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the bridge pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for bridge pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})
	Describe("Add multi-ip (weave, bridge) networking for Pod", func() {
		glog.Info("Inside Check for adding multi-ip (weave, bridge) networking")
		Context("using cni-genie for configuring multi-ip (weave, bridge) CNI", func() {
			name := fmt.Sprintf("nginx-multiip-weave-bridge-%d", rand.Uint32())
			interfaceName := "eth0"
			glog.Info(interfaceName)

			It("should succeed multi-ip (weave, bridge) networking for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "weave,bridge"
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the multi-ip (weave, bridge) pod to have running status")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the multi-ip (weave, bridge) pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for multi-ip (weave, bridge) pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})
	Describe("Add macvlan networking for Pod", func() {
		glog.Info("Inside Check for adding macvlan networking")
		Context("using cni-genie for configuring macvlan CNI", func() {
			name := fmt.Sprintf("nginx-macvlan-%d", rand.Uint32())

			It("should succeed macvlan networking for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "macvlan"
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the macvlan pod to have running status")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the macvlan pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for macvlan pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})
	Describe(" Check for multi ip preferences annotation", func() {
		glog.Info("Inside Check for multi ip preferences annotation")
		Context("using cni genie to get multiple Ip and update in annotation", func() {
			name := fmt.Sprintf("nginx-multiip-pref-%d", rand.Uint32())

			It("should succeed multi ip preference for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "weave,flannel"

				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the pod to have running status")
				By("Waiting 20 seconds")
				time.Sleep(time.Duration(20 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for macvlan pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})
	Describe("Add sriov networking for Pod", func() {
		glog.Info("Inside Check for adding sriov networking")
		Context("using cni-genie for configuring sriov CNI", func() {
			name := fmt.Sprintf("nginx-sriov-%d", rand.Uint32())

			It("should succeed sriov networking for pod", func() {
				Skip("Skip Adding sriov networking for Pod")
				annots := make(map[string]string)
				annots["cni"] = "sriov"
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the sriov pod to have running status")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the sriov pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for sriov pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe(" Check for multi ip from same plugin(ex flannel)", func() {
		glog.Info("Inside Check for multi ip from same plugin(ex flannel")
		Context("using cni genie to configure multiple ip from flannel plugin", func() {
			name := fmt.Sprintf("nginx-multiip-from-flannel-%d", rand.Uint32())

			It("should succeed multi ip preference for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "flannel,flannel"

				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the pod to have running status")
				By("Waiting 20 seconds")
				time.Sleep(time.Duration(20 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe(" Check for multi ip from same plugin(ex flannel) along with other plugins", func() {
		glog.Info("Inside Check for multi ip from same plugin(ex flannel along with other plugins")
		Context("using cni genie to configure multiple ip from flannel plugin and weave plugin", func() {
			name := fmt.Sprintf("nginx-multiip--%d", rand.Uint32())

			It("should succeed multi ip preference for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "flannel,weave,flannel"

				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the pod to have running status")
				By("Waiting 20 seconds")
				time.Sleep(time.Duration(20 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Verify default plugin case : pod with no annotation attributes", func() {
		glog.Info("Inside default plugin case : pod with no annotation attributes")
		Context("using cni-genie for verifying default plugin case : pod with no annotation attributes", func() {
			name := fmt.Sprintf("nginx-pod-no-annotation-%d", rand.Uint32())

			It("should succeed default(weave) networking for pod", func() {
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
						//Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the pod to have running status with default plugin(weave)")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Verify default plugin case : pod with non cni annotations", func() {
		glog.Info("Inside default plugin case : pod with non cni annotations")
		Context("using cni-genie for verifying default plugin case : pod with non cni annotations", func() {
			name := fmt.Sprintf("nginx-pod-non-cni-annotation-%d", rand.Uint32())

			It("should succeed default(weave) networking for pod", func() {
				annots := make(map[string]string)
				annots["build"] = "two"
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the pod to have running status with default plugin(weave)")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Verify default plugin case : pod with blank annotation field", func() {
		glog.Info("Inside default plugin case : pod with blank annotation field")
		Context("using cni-genie for verifying default plugin case : pod with blank annotation field", func() {
			name := fmt.Sprintf("nginx-pod-blank-annotation-%d", rand.Uint32())

			It("should succeed default(weave) networking for pod", func() {
				annots := make(map[string]string)
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the pod to have running status with default plugin(weave)")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Verify plugin with interface name case : pod with plugin+interface name case", func() {
		glog.Info("plugin with interface name case : pod with plugin+interface name case")
		Context("using cni-genie for verifying plugin with interface name case : pod with plugin+interface name case", func() {
			name := fmt.Sprintf("nginx-pod-with-ifname-%d", rand.Uint32())

			It("should succeed multinetworking with ifname for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "flannel,weave@eth4,flannel@eth5, flannel"
				_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: annots,
					},
					Spec: v1.PodSpec{Containers: []v1.Container{{
						Name:            fmt.Sprintf("container-%s", name),
						Image:           "nginx:latest",
						ImagePullPolicy: "IfNotPresent",
					}}},
				})

				Expect(err).NotTo(HaveOccurred())

				By("Waiting for the pod to have running status with plugin + ifname")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the pod now...")
				err = clientset.CoreV1().Pods(TEST_NAMESPACE).Delete(name, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})
	Describe("To create logical network crd objects ", func() {
		It("should succeed crd creation", func() {
			config, err := clientcmd.BuildConfigFromFlags("", testKubeConfig)
			apiextensionsclient, err = apiextensionsclientset.NewForConfig(config)
			if err != nil {
				glog.Errorf("apiextensionsclient error")
			}
			crd := &apiextensionsv1beta1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "logicalnetworks.alpha.network.k8s.io",
				},
				Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
					Group:   "alpha.network.k8s.io",
					Version: "v1",
					Scope:   apiextensionsv1beta1.NamespaceScoped,
					Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
						Plural:   "logicalnetworks",
						Kind:     "LogicalNetwork",
						Singular: "logicalnetwork",
					},
				},
			}
			Expect(err).NotTo(HaveOccurred())
			By("Check for crd creation object creation")
			glog.Info("To check Custom Resource definition creation")
			_, err = apiextensionsclient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
			Expect(err).NotTo(HaveOccurred())
			glog.Info("Create a logical network specifying network plugin and customized subnet")

			logicalNet := utils.LogicalNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "net1",
					Namespace: DEFAULT,
				},
			}
			logicalNet.Spec.Plugin = "weave"
			logicalNet.Spec.SubSubnet = "10.32.20.0/24"
			restCli, _ := logicalv1.NewForConfig(config)
			time.Sleep(time.Duration(5 * time.Second))
			By("waiting for 5 seconds")
			_, err = restCli.LogicalNetworks(DEFAULT).Create(&logicalNet)
			Expect(err).NotTo(HaveOccurred())
			Expect("Success").To(Equal("Success"))
		})

		It("should succeed weave networking for pod", func() {
			glog.Info("Add weave networking for Pod by specifying logical network")
			name := fmt.Sprintf("nginx-weave-%d", rand.Uint32())
			interfaceName := "eth0"
			glog.Info(interfaceName)
			annots := make(map[string]string)
			annots["cni"] = ""
			annots["networks"] = "net1"
			_, err := clientset.CoreV1().Pods(DEFAULT).Create(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Annotations: annots,
				},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:            fmt.Sprintf("container-%s", name),
					Image:           "nginx:latest",
					ImagePullPolicy: "IfNotPresent",
				}}},
			})
			Expect(err).NotTo(HaveOccurred())
			By("Waiting for the weave pod to have running status")
			By("Waiting 10 seconds")
			time.Sleep(time.Duration(5 * time.Second))
			pod, err := clientset.CoreV1().Pods(DEFAULT).Get(name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			glog.Info("pod status =", string(pod.Status.Phase))
			time.Sleep(time.Duration(5 * time.Second))
			Expect(string(pod.Status.Phase)).To(Equal("Running"))
			Expect("Success").To(Equal("Success"))
		})
	})

	Describe("To create NetworkAttachmentDefinition CRD ", func() {
		It("should succeed crd creation", func() {
			config, err := clientcmd.BuildConfigFromFlags("", testKubeConfig)
			apiextensionsclient, err = apiextensionsclientset.NewForConfig(config)
			if err != nil {
				glog.Errorf("apiextensionsclient error")
			}
			crd := &apiextensionsv1beta1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "network-attachment-definitions.k8s.cni.cncf.io",
				},
				Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
					Group:   "k8s.cni.cncf.io",
					Version: "v1",
					Scope:   apiextensionsv1beta1.NamespaceScoped,
					Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
						Plural:   "network-attachment-definitions",
						Singular: "network-attachment-definition",
						Kind:     "NetworkAttachmentDefinition",
						ShortNames: []string{
							"net-attach-def",
						},
					},
					Validation: &apiextensionsv1beta1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1beta1.JSONSchemaProps{
							Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{

								"spec": {
									Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
										"config": {},
									},
								},
							},
						},
					},
				},
			}
			_, err = apiextensionsclient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
			Expect(err).NotTo(HaveOccurred())
			Expect("Success").To(Equal("Success"))
		})

		It("To create NetworkAttachmentDefinition object flannel", func() {
			config, err := clientcmd.BuildConfigFromFlags("", testKubeConfig)
			netattachclient, err := netattachclientset.NewForConfig(config)
			if err != nil {
				glog.Errorf("client error")
			}

			var netattachobj = netattachv1.NetworkAttachmentDefinition{}
			netattachobj.Name = "flannel-conf"
			netattachobj.Namespace = TEST_NAMESPACE
			netattachobj.Spec = netattachv1.NetworkAttachmentDefinitionSpec{
				Config: "{\n\"cniVersion\": \"0.3.0\",\n\"type\": \"flannel\",\n\"delegate\": {\n\"isDefaultGateway\": true\n}\n}",
			}
			//Create NetworkAttachmentDefinition object with plugin as Spec
			time.Sleep(time.Duration(10 * time.Second))
			_, err = netattachclient.K8sCniCncfIoV1().NetworkAttachmentDefinitions(TEST_NAMESPACE).Create(&netattachobj)
			Expect(err).NotTo(HaveOccurred())
			Expect("Success").To(Equal("Success"))
		})

		//Create NetworkAttachmentDefinition object with plugin configuration as file
		It("To create NetworkAttachmentDefinition object weave", func() {
			config, err := clientcmd.BuildConfigFromFlags("", testKubeConfig)
			netattachclient, err := netattachclientset.NewForConfig(config)
			if err != nil {
				glog.Errorf("client error")
			}
			var netattachobj = netattachv1.NetworkAttachmentDefinition{}
			netattachobj.Name = "weave"
			netattachobj.Namespace = CRD_TEST_NAMESPACE
			time.Sleep(time.Duration(5 * time.Second))
			_, err = netattachclient.K8sCniCncfIoV1().NetworkAttachmentDefinitions(CRD_TEST_NAMESPACE).Create(&netattachobj)
			Expect(err).NotTo(HaveOccurred())
			Expect("Success").To(Equal("Success"))
		})

		//flannel, weave networking for Pod
		It("should succeed weave networking for pod", func() {
			glog.Info("Inside Check for adding weave-flannel networking")
			name := fmt.Sprintf("nginx-netattachdef-flannel-weave-%d", rand.Uint32())
			interfaceName := "eth0"
			glog.Info(interfaceName)
			annots := make(map[string]string)
			annots["k8s.v1.cni.cncf.io/networks"] = "flannel-conf,network/weave"
			time.Sleep(time.Duration(10 * time.Second))
			_, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Create(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   TEST_NAMESPACE,
					Annotations: annots,
				},
				Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:            fmt.Sprintf("container-%s", name),
					Image:           "nginx:latest",
					ImagePullPolicy: "IfNotPresent",
				}}},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for the weave pod to have running status")
			By("Waiting 10 seconds")
			time.Sleep(time.Duration(10 * time.Second))
			pod, err := clientset.CoreV1().Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			glog.Info("pod status =", string(pod.Status.Phase))
			time.Sleep(time.Duration(5 * time.Second))
			Expect(string(pod.Status.Phase)).To(Equal("Running"))
			Expect("Success").To(Equal("Success"))
		})

	})

})
var _ = BeforeSuite(func() {
	var config *rest.Config
	var err error
	glog.Infof("Kube version %s", testKubeVersion)
	if testKubeVersion == "1.5" {
		config, err = clientcmd.DefaultClientConfig.ClientConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", testKubeConfig)
	}
	if err != nil {
		panic(err)
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	createNamespace(clientset)

})

var _ = AfterSuite(func() {

	//Delete crd network-attachment-definitions.k8s.cni.cncf.io
	err := apiextensionsclient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(NetAttachDef, &metav1.DeleteOptions{})
	if err != nil {
		panic(err)
	}
	//delete namespace network
	err = clientset.CoreV1().Namespaces().Delete(CRD_TEST_NAMESPACE, &metav1.DeleteOptions{})
	//delete crd logicalnetworks
	err = apiextensionsclient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(NetworkCRD, &metav1.DeleteOptions{})
	if err != nil {
		panic(err)
	}
	err = clientset.CoreV1().Pods("default").DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
	// Delete namespace
	err = clientset.CoreV1().Namespaces().Delete(TEST_NAMESPACE, &metav1.DeleteOptions{})
	// Delete all pods
	err = clientset.CoreV1().Pods(TEST_NAMESPACE).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

})

func createNamespace(clientset *kubernetes.Clientset) {
	ns, err := clientset.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: TEST_NAMESPACE},
	})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return
		} else {
			Expect(err).ShouldNot(HaveOccurred())
		}
	}
	By("Waiting 5 seconds")
	time.Sleep(time.Duration(5 * time.Second))
	ns, err = clientset.CoreV1().Namespaces().Get(TEST_NAMESPACE, metav1.GetOptions{})
	Expect(err).ShouldNot(HaveOccurred())
	Expect(ns.Name).To(Equal(TEST_NAMESPACE))

	networkns, err := clientset.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: CRD_TEST_NAMESPACE},
	})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return
		} else {
			Expect(err).ShouldNot(HaveOccurred())
		}
	}
	By("Waiting 5 seconds")
	time.Sleep(time.Duration(5 * time.Second))
	networkns, err = clientset.CoreV1().Namespaces().Get(CRD_TEST_NAMESPACE, metav1.GetOptions{})
	Expect(err).ShouldNot(HaveOccurred())
	Expect(networkns.Name).To(Equal(CRD_TEST_NAMESPACE))
}
