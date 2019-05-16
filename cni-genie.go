//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"fmt"
	"github.com/cni-genie/CNI-Genie/genie"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
	"os"
	"runtime"
)

func init() {
	// This ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func cmdAdd(args *skel.CmdArgs) error {
	fmt.Fprintf(os.Stderr, "CNI Genie cmdAdd = %v\n", string(args.StdinData))

	conf, err := genie.ParseCNIConf(args.StdinData)
	if err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}
	gc, err := genie.NewGenieController(conf)
	if err != nil {
		return err
	}
	cniArgs := genie.PopulateCNIArgs(args)
	fmt.Fprintf(os.Stderr, "CNI Genie Add IP address\n")
	result, ipamErr := gc.AddPodNetwork(cniArgs, conf)
	if ipamErr != nil || nil == result {
		return fmt.Errorf("CNI Genie Add IP internal error: %v, result: %s", ipamErr, result)
	}

	fmt.Fprintf(os.Stderr, "CNI Genie End result= %s\n", result)
	return types.PrintResult(result, conf.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	fmt.Fprintf(os.Stderr, "CNI Genie cmdDel = %v\n", string(args.StdinData))

	conf, err := genie.ParseCNIConf(args.StdinData)
	if err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}
	gc, err := genie.NewGenieController(conf)
	if err != nil {
		return err
	}
	cniArgs := genie.PopulateCNIArgs(args)
	fmt.Fprintf(os.Stderr, "CNI Genie releasing IP address\n")
	ipamErr := gc.DeletePodNetwork(cniArgs, conf)
	if ipamErr != nil {
		return fmt.Errorf("CNI Genie release IP internal error: %v", ipamErr)
	}

	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	fmt.Fprintf(os.Stderr, "CNI Genie cmdCheck = %v\n", string(args.StdinData))

	return nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, "CNI Genie plugin")
}
