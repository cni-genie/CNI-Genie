package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCNIGenie(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CNIGenie Suite")
}
