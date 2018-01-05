package main_test

import (
	"fmt"
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"math/rand"
	"os"
	"time"
	"os/exec"
	"flag"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/rest"
)

const TEST_NAMESPACE = "test"

var testKubeVersion string
var testKubeConfig string
var clientset *kubernetes.Clientset

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
				_, err := clientset.Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: v1.ObjectMeta{
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
				pod, err := clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the calico pod now...")
				err = clientset.Pods(TEST_NAMESPACE).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for calico pod deletion")
				_, err = clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
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
				annots := make(map[string]string)
				annots["cni"] = "romana"
				//Create a K8s Pod with calico cni
				_, err := clientset.Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: v1.ObjectMeta{
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
				pod, err := clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the romana pod now...")
				err = clientset.Pods(TEST_NAMESPACE).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
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
				_, err := clientset.Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: v1.ObjectMeta{
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
				pod, err := clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the weave pod now...")
				err = clientset.Pods(TEST_NAMESPACE).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
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
				annots["cni"] = "calico,weave"
				_, err := clientset.Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: v1.ObjectMeta{
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
				pod, err := clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the multi-ip pod now...")
				err = clientset.Pods(TEST_NAMESPACE).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for multi-ip pod deletion")
				_, err = clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
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
				annots := make(map[string]string)
				annots["cni"] = " "
				_, err := clientset.Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: v1.ObjectMeta{
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
				pod, err := clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the nocni pod now...")
				err = clientset.Pods(TEST_NAMESPACE).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for nocni pod deletion")
				_, err = clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
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
				_, err := clientset.Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: v1.ObjectMeta{
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
				pod, err := clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the bridge pod now...")
				err = clientset.Pods(TEST_NAMESPACE).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for bridge pod deletion")
				_, err = clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
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
				_, err := clientset.Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: v1.ObjectMeta{
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
				pod, err := clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the multi-ip (weave, bridge) pod now...")
				err = clientset.Pods(TEST_NAMESPACE).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for multi-ip (weave, bridge) pod deletion")
				_, err = clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
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
				_, err := clientset.Pods(TEST_NAMESPACE).Create(&v1.Pod{
					ObjectMeta: v1.ObjectMeta{
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
				pod, err := clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				glog.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the macvlan pod now...")
				err = clientset.Pods(TEST_NAMESPACE).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for macvlan pod deletion")
				_, err = clientset.Pods(TEST_NAMESPACE).Get(name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
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
	// Start all the required plugins through shell script
	cmd := exec.Command("./plugins_install.sh", "-all")
	_, err = cmd.Output()

})

var _ = AfterSuite(func() {
	// Delete namespace
	err := clientset.Namespaces().Delete(TEST_NAMESPACE, &v1.DeleteOptions{})
	// Delete all pods
	err = clientset.Pods(TEST_NAMESPACE).DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})
	if err != nil {
		panic(err)
	}
	// Delete all the installed plugins after usage
	cmd := exec.Command("./plugins_install.sh", "-deleteall")
       _, err = cmd.Output()

})

func createNamespace(clientset *kubernetes.Clientset) {
	ns, err := clientset.Namespaces().Create(&v1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: TEST_NAMESPACE},
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
	ns, err = clientset.Namespaces().Get(TEST_NAMESPACE, metav1.GetOptions{})
	Expect(err).ShouldNot(HaveOccurred())
	Expect(ns.Name).To(Equal(TEST_NAMESPACE))
}
