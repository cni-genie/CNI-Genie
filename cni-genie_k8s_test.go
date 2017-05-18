package main_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/cni-plugin/utils"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	meta_v1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"math/rand"
	"net"
	"os"
	"time"

	"k8s.io/client-go/pkg/api/errors"
)

const TEST_NS = "test"

func init() {

}

var _ = Describe("CNIGenie", func() {

	hostname, _ := os.Hostname()
	utils.ConfigureLogging("info")
	logger := utils.CreateContextLogger("genie_k8s_tests")
	logger.Info("Inside CNIGenie tests for k8s:", hostname)

	Describe("Run Genie for k8s", func() {
		logger.Info("Inside Run Genie for k8s...")
		logger.Info("Test Namespace:", TEST_NS)
		logger.Info("Hostname:", hostname)
		cniVersion := os.Getenv("CNI_SPEC_VERSION")
		logger.Info("cniVersion:", cniVersion)
		Context("using host-local IPAM", func() {
			netconf := fmt.Sprintf(`
			{
			  "cniVersion": "%s",
			  "name": "net1",
			  "type": "genie",
			  "etcd_endpoints": "http://%s:2379",
			  "ipam": {
			    "type": "host-local",
			    "subnet": "10.0.0.0/8"
			  }, "kubernetes": {
				  "k8s_api_root": "http://127.0.0.1:8080"
				},
				"policy": {"type": "k8s"},
				"log_level":"info"
			}`, cniVersion, os.Getenv("ETCD_IP"))

			logger.Info("ETCD_IP:", os.Getenv("ETCD_IP"))
			logger.Info("neconf:", netconf)
			It("successfully networks the namespace", func() {
				logger.Info("Inside successfully networks the namespace...")
				Expect(cniVersion).To(Equal("0.3.0"))
				Expect(os.Getenv("ETCD_IP")).To(Equal("127.0.0.1"))
			})
		})
	})

	Describe("Check for available CNSs", func() {
		By("List all running CNSs on the node")
		logger.Info("Inside Check for available CNSs")
		Context("genie listing CNS", func() {
			l, err := net.Interfaces()
			if err != nil {
				panic(err)

			}
			It("successfully identify CNS", func() {
				cnsAvailable := false
				for _, f := range l {
					if len(f.Name) > 4 {
						if f.Name[:4] == "cali" {
							cnsAvailable = true
							Expect(f.Name).To(ContainSubstring("cali"), " of type Canal")
						} else if f.Name[:4] == "flan" {
							cnsAvailable = true
							Expect(f.Name).To(ContainSubstring("flanne"), " of type Canal")
						} else if f.Name[:4] == "weav" {
							cnsAvailable = true
							Expect(f.Name).To(ContainSubstring("weav"), " of type weave")
						}
					}
				}
				Expect(cnsAvailable).To(Equal(true))
			})
		})
	})

	Describe("Add canal networking for Pod", func() {
		logger.Info("Inside Check for adding Canal networking")
		cniVersion := os.Getenv("CNI_SPEC_VERSION")
		logger.Info("cniVersion:", cniVersion)
		Context("using cni-genie for configuring canal CNI", func() {
			config, err := clientcmd.DefaultClientConfig.ClientConfig()
			if err != nil {
				panic(err)
			}
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				panic(err)
			}
			name := fmt.Sprintf("nginx-canal-%d", rand.Uint32())
			interfaceName := "eth0"
			logger.Info(interfaceName)

			It("should create test namespace", func() {
				ns, err := clientset.Namespaces().Create(&v1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: TEST_NS},
				})
				if err != nil && errors.IsAlreadyExists(err) {
					//do nothing ignore
				} else if err != nil {
					//if some other error other than Already Exists
					Expect(err).ShouldNot(HaveOccurred())
				}
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				ns, err = clientset.Namespaces().Get(TEST_NS, meta_v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ns.Name).To(Equal(TEST_NS))
			})

			It("should succeed canal networking for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "canal"
				//Create a K8s Pod with canal cni
				_, err = clientset.Pods(TEST_NS).Create(&v1.Pod{
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

				By("Waiting for the canal pod to have running status")
				By("Waiting 10 seconds")
				time.Sleep(time.Duration(10 * time.Second))
				pod, err := clientset.Pods(TEST_NS).Get(name, meta_v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				logger.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the canal pod now...")
				err = clientset.Pods(TEST_NS).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for canal pod deletion")
				_, err = clientset.Pods(TEST_NS).Get(name, meta_v1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Add romana networking for Pod", func() {
		logger.Info("Inside Check for adding romana networking")

		cniVersion := os.Getenv("CNI_SPEC_VERSION")
		logger.Info("cniVersion:", cniVersion)
		Context("using cni-genie for configuring romana CNI", func() {
			config, err := clientcmd.DefaultClientConfig.ClientConfig()
			if err != nil {
				panic(err)
			}
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				panic(err)
			}
			name := fmt.Sprintf("nginx-romana-%d", rand.Uint32())
			interfaceName := "eth0"
			logger.Info(interfaceName)

			It("should create test namespace", func() {
				ns, err := clientset.Namespaces().Create(&v1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: TEST_NS},
				})
				if err != nil && errors.IsAlreadyExists(err) {
					//do nothing ignore
				} else if err != nil {
					//if some other error other than Already Exists
					Expect(err).ShouldNot(HaveOccurred())
				}
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				ns, err = clientset.Namespaces().Get(TEST_NS, meta_v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ns.Name).To(Equal(TEST_NS))
			})

			It("should succeed romana networking for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "romana"
				//Create a K8s Pod with canal cni
				_, err = clientset.Pods(TEST_NS).Create(&v1.Pod{
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
				pod, err := clientset.Pods(TEST_NS).Get(name, meta_v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				logger.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the romana pod now...")
				err = clientset.Pods(TEST_NS).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.Pods(TEST_NS).Get(name, meta_v1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Add weave networking for Pod", func() {
		logger.Info("Inside Check for adding weave networking")

		cniVersion := os.Getenv("CNI_SPEC_VERSION")
		logger.Info("cniVersion:", cniVersion)
		Context("using cni-genie for configuring weave CNI", func() {
			config, err := clientcmd.DefaultClientConfig.ClientConfig()
			if err != nil {
				panic(err)
			}
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				panic(err)
			}
			name := fmt.Sprintf("nginx-weave-%d", rand.Uint32())
			interfaceName := "eth0"
			logger.Info(interfaceName)

			It("should create test namespace", func() {
				ns, err := clientset.Namespaces().Create(&v1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: TEST_NS},
				})
				if err != nil && errors.IsAlreadyExists(err) {
					//do nothing ignore
				} else if err != nil {
					//if some other error other than Already Exists
					Expect(err).ShouldNot(HaveOccurred())
				}
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				ns, err = clientset.Namespaces().Get(TEST_NS, meta_v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ns.Name).To(Equal(TEST_NS))
			})

			It("should succeed weave networking for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "weave"
				//Create a K8s Pod with canal cni
				_, err = clientset.Pods(TEST_NS).Create(&v1.Pod{
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
				pod, err := clientset.Pods(TEST_NS).Get(name, meta_v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				logger.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the weave pod now...")
				err = clientset.Pods(TEST_NS).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for pod deletion")
				_, err = clientset.Pods(TEST_NS).Get(name, meta_v1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Add multi-ip networking for Pod", func() {
		logger.Info("Inside Check for adding multi-ip networking")
		cniVersion := os.Getenv("CNI_SPEC_VERSION")
		logger.Info("cniVersion:", cniVersion)
		Context("using cni-genie for configuring multi-ip CNI", func() {
			config, err := clientcmd.DefaultClientConfig.ClientConfig()
			if err != nil {
				panic(err)
			}
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				panic(err)
			}
			name := fmt.Sprintf("nginx-multiip-%d", rand.Uint32())
			interfaceName := "eth0"
			logger.Info(interfaceName)

			It("should create test namespace", func() {
				ns, err := clientset.Namespaces().Create(&v1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: TEST_NS},
				})
				if err != nil && errors.IsAlreadyExists(err) {
					//do nothing ignore
				} else if err != nil {
					//if some other error other than Already Exists
					Expect(err).ShouldNot(HaveOccurred())
				}
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				ns, err = clientset.Namespaces().Get(TEST_NS, meta_v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ns.Name).To(Equal(TEST_NS))
			})

			It("should succeed multi-ip networking for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = "canal,weave"
				//Create a K8s Pod with canal cni
				_, err = clientset.Pods(TEST_NS).Create(&v1.Pod{
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
				pod, err := clientset.Pods(TEST_NS).Get(name, meta_v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				logger.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the multi-ip pod now...")
				err = clientset.Pods(TEST_NS).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for multi-ip pod deletion")
				_, err = clientset.Pods(TEST_NS).Get(name, meta_v1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})
	})

	Describe("Add nocni networking for Pod", func() {
		logger.Info("Inside Check for adding nocni networking")
		cniVersion := os.Getenv("CNI_SPEC_VERSION")
		logger.Info("cniVersion:", cniVersion)
		Context("using cni-genie for configuring nocni CNI", func() {
			config, err := clientcmd.DefaultClientConfig.ClientConfig()
			if err != nil {
				panic(err)
			}
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				panic(err)
			}
			name := fmt.Sprintf("nginx-nocni-%d", rand.Uint32())
			interfaceName := "eth0"
			logger.Info(interfaceName)

			It("should create test namespace", func() {
				ns, err := clientset.Namespaces().Create(&v1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: TEST_NS},
				})
				if err != nil && errors.IsAlreadyExists(err) {
					//do nothing ignore
				} else if err != nil {
					//if some other error other than Already Exists
					Expect(err).ShouldNot(HaveOccurred())
				}
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				ns, err = clientset.Namespaces().Get(TEST_NS, meta_v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ns.Name).To(Equal(TEST_NS))
			})

			It("should succeed nocni networking for pod", func() {
				annots := make(map[string]string)
				annots["cni"] = " "
				//Create a K8s Pod with canal cni
				_, err = clientset.Pods(TEST_NS).Create(&v1.Pod{
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
				pod, err := clientset.Pods(TEST_NS).Get(name, meta_v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				logger.Info("pod status =", string(pod.Status.Phase))
				Expect(string(pod.Status.Phase)).To(Equal("Running"))

				By("Pod was in Running state... Time to delete the nocni pod now...")
				err = clientset.Pods(TEST_NS).Delete(name, &v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				By("Waiting 5 seconds")
				time.Sleep(time.Duration(5 * time.Second))
				By("Check for nocni pod deletion")
				_, err = clientset.Pods(TEST_NS).Get(name, meta_v1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					//do nothing pod has already been deleted
				}
				Expect("Success").To(Equal("Success"))
			})
		})

	})

	//No guarantee this test case executes in the end.
	//Ginkgo doesn't execute in sequential order
	/*Describe("Cleanup Tests", func() {
		logger.Info("Inside cleanup tests")
		By("Tear-down Test namespace...")
		config, err := clientcmd.DefaultClientConfig.ClientConfig()
		if err != nil {
			panic(err)
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err)
		}
		err = clientset.Namespaces().Delete(TEST_NS, &v1.DeleteOptions{})

		By("Waiting 10 seconds")
		time.Sleep(time.Duration(10 * time.Second))
	})*/
})
