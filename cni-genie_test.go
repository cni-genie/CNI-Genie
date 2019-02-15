package main

import (
	"encoding/json"
	"errors"
	"github.com/cni-genie/CNI-Genie/utils"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"strings"
	"testing"
)

func TestCmdAdd(t *testing.T) {
	conf := utils.GenieConf{
		NetConf: types.NetConf{
			Name: "k8s-pod-network",
			Type: "genie",
		},
	}

	bytes, err := json.Marshal(&conf)
	if err != nil {
		t.Errorf(err.Error())
	}

	tests := []struct {
		args        *skel.CmdArgs
		expectError error
	}{
		{
			&skel.CmdArgs{StdinData: bytes},
			errors.New("Error building kubernetes client"),
		},
	}

	for i := range tests {
		err := cmdAdd(tests[i].args)
		if !strings.Contains(err.Error(), tests[i].expectError.Error()) {
			t.Errorf("Error executing test: expected error: %v, got error: %v", tests[i].expectError, err)
		}
	}
}

func TestCmdDel(t *testing.T) {
	conf := utils.GenieConf{
		NetConf: types.NetConf{
			Name: "k8s-pod-network",
			Type: "genie",
		},
	}

	bytes, err := json.Marshal(&conf)
	if err != nil {
		t.Errorf(err.Error())
	}

	tests := []struct {
		args        *skel.CmdArgs
		expectError error
	}{
		{
			&skel.CmdArgs{StdinData: bytes},
			errors.New("Error building kubernetes client"),
		},
	}

	for i := range tests {
		err := cmdDel(tests[i].args)
		if !strings.Contains(err.Error(), tests[i].expectError.Error()) {
			t.Errorf("Error executing test: expected error: %v, got error: %v", tests[i].expectError, err)
		}
	}
}
