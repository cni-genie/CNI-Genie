package interfaces

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"net"
	"os"
	"strings"
)

type FakeIo struct {
	Files []string
}

type FakeInvoke struct {
	Result types.Result
	Error  error
}

type FakeCni struct {
	InstalledPlugins []string
	Files            []string
	config           map[string]string
}

var config map[string]string = map[string]string{
	"weave": `{
    "cniVersion": "0.3.0",
    "name": "weave",
    "plugins": [
        {
            "name": "weave",
            "type": "weave"
        }
    ]
}`,
	"flannel": `{
  "name": "cbr0",
  "type": "flannel",
  "delegate": {
    "isDefaultGateway": true
  }
}`,
	"bridge": `{
  "name": "mybridgenet",
  "type": "bridge",
  "ipam": {
    "type": "host-local"
  }
}`,
	"macvlan": `{
  "name": "macvlannet",
  "type": "macvlan",
  "ipam": {
    "type": "host-local"
  }
}`,
}

var ip map[string]net.IPNet

func (fi *FakeIo) ReadFile(file string) ([]byte, error) {
	if file == "" {
		return nil, errors.New("Invalid file path")
	}
	if c, ok := config[parseFileName(file)]; ok {
		return []byte(c), nil
	} else {
		return nil, errors.New("File not present")
	}
}

func (fi *FakeIo) ReadDir(dir string) ([]os.FileInfo, error) {
	if dir == "" {
		return nil, errors.New("Invalid directory path")
	}
	return nil, nil
}

func (fi *FakeIo) CreateFile(filePath string, bytes []byte, perm os.FileMode) error {
	if filePath == "" {
		return errors.New("Invalid file path")
	}
	config[parseFileName(filePath)] = string(bytes)
	fi.Files = append(fi.Files, filePath)
	return nil
}

func SetIp(cni []string) {
	ip = map[string]net.IPNet{}
	var cnt uint8 = 1
	for p := range config {
		switch p {
		case "flannel":
			ip[p] = net.IPNet{IP: net.IPv4(byte(10), byte(244), byte(0), byte(1))}
		case "weave":
			ip[p] = net.IPNet{IP: net.IPv4(byte(10), byte(32), byte(0), byte(1))}
		case "bridge":
			ip[p] = net.IPNet{IP: net.IPv4(byte(10), byte(10), byte(0), byte(1))}
		case "calico":
			ip[p] = net.IPNet{IP: net.IPv4(byte(192), byte(168), byte(0), byte(1))}
		case "macvlan":
			ip[p] = net.IPNet{IP: net.IPv4(byte(10), byte(10), byte(0), byte(1))}
		default:
			ip[p] = net.IPNet{IP: net.IPv4(byte(0), byte(cnt), byte(0), byte(1))}
			cnt++
		}
	}

	for _, c := range cni {
		if ip[c].IP == nil {
			ip[c] = net.IPNet{IP: net.IPv4(byte(10), byte(cnt), byte(0), byte(1))}
			cnt++
		}
	}
}

func buildResult(cni string) types.Result {
	res := current.Result{}
	ipaddr := ip[cni].IP
	fmt.Println("ip[cni]:", ipaddr)
	res.IPs = []*current.IPConfig{{Version: "4", Address: net.IPNet{IP: ipaddr, Mask: ipaddr.DefaultMask()}}}
	i := ipaddr.To4()
	i[3]++
	ip[cni] = net.IPNet{IP: i}

	return &res
}

func (i *FakeInvoke) InvokeExecAdd(config *libcni.NetworkConfigList, rtConf *libcni.RuntimeConf) (types.Result, error) {
	return buildResult(config.Plugins[0].Network.Type), nil
}

func (i *FakeInvoke) InvokeExecDel(config *libcni.NetworkConfigList, rtConf *libcni.RuntimeConf) error {
	return i.Error
}

var fakeConfig *CNIConfig = &CNIConfig{
	CNI: &FakeCni{},
	RW:  &FakeIo{},
}

func (c *FakeCni) ConfListFromFile(file string) (*libcni.NetworkConfigList, error) {
	return fakeConfig.ParseCNIConfFromBytes([]byte(config[parseFileName(file)]))
}

func (c *FakeCni) ConfListFromBytes(bytes []byte) (*libcni.NetworkConfigList, error) {
	return libcni.ConfListFromBytes(bytes)
}

func (c *FakeCni) ConfFromFile(file string) (*libcni.NetworkConfig, error) {
	return c.ConfFromBytes([]byte(config[parseFileName(file)]))
}

func (c *FakeCni) ConfFromBytes(bytes []byte) (*libcni.NetworkConfig, error) {
	return (&Cni{}).ConfFromBytes(bytes)
}

func (c *FakeCni) ConfListFromConf(conf *libcni.NetworkConfig) (*libcni.NetworkConfigList, error) {
	return (&Cni{}).ConfListFromConf(conf)
}

func (c *FakeCni) ConfListFromConfBytes(confBytes []byte) (*libcni.NetworkConfigList, error) {
	return (&Cni{}).ConfListFromConfBytes(confBytes)
}

func (c *FakeCni) ConfFiles(dir string, ext []string) ([]string, error) {
	var extn string
	files := make([]string, 0, len(config))
	for plugin, conf := range config {
		obj := map[string]interface{}{}
		_ = json.Unmarshal([]byte(conf), &obj)
		if _, ok := obj["plugins"]; ok {
			extn = ".conflist"
		} else {
			extn = ".conf"
		}
		files = append(files, DefaultNetDir+"10-"+plugin+extn)
	}
	c.config = config
	defaultConf := `{"name": "%s", "type": "%s"}`
	for _, plg := range c.InstalledPlugins {
		if c.config[plg] == "" {
			c.config[plg] = fmt.Sprintf(defaultConf, plg, plg)
			fmt.Println("c.config[plg]:", c.config[plg])
			files = append(files, DefaultNetDir+"10-"+plg+".conf")
		}
	}
	c.Files = files
	return files, nil
}

func parseFileName(name string) string {
	return name[strings.Index(name, "-")+1 : strings.Index(name, ".conf")]
}
