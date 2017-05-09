package main_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	"os"
)

func init() {
	var err error
	if err != nil {
		panic(err)
	}
}

var _ = Describe("CalicoCni", func() {
	hostname, _ := os.Hostname()

	cniVersion := os.Getenv("CNI_SPEC_VERSION")

	Describe("Run Calico CNI plugin", func() {
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
			  }
			}`, cniVersion, os.Getenv("ETCD_IP"))

			It("successfully networks the namespace", func() {
			})
		})
	})
})