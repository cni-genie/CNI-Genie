package main


import (
	"bytes"
	"fmt"
	"os/exec"
	. "github.com/projectcalico/cni-plugin/utils"
	"os"
	//"encoding/json"
	"bufio"
)

func main() {
	fmt.Println("**** Start cni_genie_main ****")
	stdout := &bytes.Buffer{}

	var envs [5]string
	envs[0] = "798dea77ad837e767685fca659872619a7868f20f0f8714eb666687013d8ad52"
	envs[1] = "/proc/24640/ns/net"
	envs[2] = "eth0"
	//envs[3] = "IgnoreUnknown=1;IgnoreUnknown=1;K8S_POD_NAMESPACE=default;K8S_POD_NAME=nginx-test;K8S_POD_INFRA_CONTAINER_ID=b7886a63231dbf5b8b219398df74db535e422ed4ec976636ee751f6aa9331627"
	envs[3] = ""
	envs[4] = "/opt/cni/bin:/opt/calico/bin"
	//conf := NetConf{}
	//conf.Type = "calico"
	//conf.IPAM.Type = "calico-ipam"

	//var stdinData []byte;

	//stdinData,_ = json.Marshal(&conf)

	env := os.Environ()

	env = append(env,
		"CNI_COMMAND=ADD",
		"CNI_CONTAINERID="+envs[0],
		"CNI_NETNS="+envs[1],
		"CNI_ARGS="+envs[3],
		"CNI_IFNAME="+envs[2],
		"CNI_PATH="+envs[4])

	file, err := os.Open("/etc/cni/net.d/10-calico.conf")
	if err != nil {
		fmt.Println("error=", err)
	}

	c := exec.Cmd{
		Env: env,
		Path:   "/opt/cni/bin",
		//Args:   []string{"/opt/cni/bin/calico"},
		Stdin:  bufio.NewReader(file),
		Stdout: stdout,
		//Dir: "/opt/cni/bin",
		//Stderr: e.Stderr,
	}

	cOutput, _ := c.Output()
	fmt.Println("cmd output=", cOutput)

	err = c.Run()

	if err != nil {
		fmt.Println("error=", err)
	}

	fmt.Println("*** result=", string(stdout.Bytes()))


	//cmd2 := exec.Command("/opt/cni/bin/calico")

	//path, err := exec.LookPath("/opt/cni/bin/calico")

	//if err !=nil {
	//	panic(err)
	//}

	//args := []string{file.}

	fmt.Println("**** Start cni_genie_main ****")
}
