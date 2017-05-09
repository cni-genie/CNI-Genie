package main_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	"os"
	"github.com/projectcalico/cni-plugin/utils"
)

func init() {

}

var _ = Describe("CNIGenie", func() {
	hostname, _ := os.Hostname()
	utils.ConfigureLogging("info")
	fmt.Println("Inside CNIGenie tests for k8s ***")

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

	Describe("Check for available CNSs", func() {
		logger := utils.CreateContextLogger("avaialbe_CNS")
		logger.Info("Inside Check for available CNSs")

	})

	Describe("Add canal networking for Pod", func() {
		logger := utils.CreateContextLogger("genie_k8s_tests")
		logger.Info("Inside Check for adding Canal networking")

	})

	Describe("Add weave networking for Pod", func() {
		logger := utils.CreateContextLogger("genie_k8s_tests")
		logger.Info("Inside Check for adding weave networking")

	})

	Describe("Add nocni networking for Pod", func() {
		logger := utils.CreateContextLogger("genie_k8s_tests")
		logger.Info("Inside Check for adding nocni networking")

	})
})