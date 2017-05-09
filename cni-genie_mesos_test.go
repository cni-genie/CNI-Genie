package main_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	"github.com/projectcalico/cni-plugin/utils"
)

func init() {

}

var _ = Describe("CNIGenie", func() {
	utils.ConfigureLogging("info")
	fmt.Println("Inside CNIGenie mesos tests")
	Describe("Run Genie for mesos", func() {
		logger := utils.CreateContextLogger("genie_mesos_tests")
		logger.Info("Inside Run Genie for mesos...")
	})
})