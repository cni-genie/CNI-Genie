package main_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	"os"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/rand"
	"time"
	"github.com/projectcalico/cni-plugin/utils"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

var _ = Describe("CNIGenie", func() {
	hostname, _ := os.Hostname()
	utils.ConfigureLogging("info")
	fmt.Println("Inside CNIGenie test case ***")
	Describe("Run Genie for k8s", func() {
		logger := utils.CreateContextLogger("genie_k8s_tests")
		logger.Info("Inside Run Genie for k8s...")
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
			})
		})
	})

	Describe("Run Genie for mesos", func() {
		logger := utils.CreateContextLogger("genie_mesos_tests")
		logger.Info("Inside Run Genie for mesos...")
	})
})