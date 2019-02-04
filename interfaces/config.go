package interfaces

import (
	"fmt"
	"github.com/containernetworking/cni/libcni"
	"k8s.io/apimachinery/pkg/util/json"
	"os"
	"strings"
)

const (
	// ConfFilePermission specifies the default permission for conf file
	ConfFilePermission os.FileMode = 0644
	DefaultNetDir                  = "/etc/cni/net.d/"
	// DefaultPluginDir specifies the default directory path for cni binary files
	DefaultPluginDir = "/opt/cni/bin/"
)

type CNIConfig struct {
	RW
	CNI
	NetDir string
	BinDir string
	Files  []string
}

func (c *CNIConfig) GetInstalledPlugins() ([]string, error) {
	binaries, err := c.ReadDir(c.BinDir)
	if err != nil {
		return nil, err
	}

	plugins := make([]string, 0, len(binaries))
	for i := range binaries {
		plugins = append(plugins, binaries[i].Name())
	}

	return plugins, nil
}

// placeConfFile creates a conf file in the specified directory path
func (c *CNIConfig) CreateConfFile(cniName string, bytes []byte) error {
	confFile := fmt.Sprintf(c.NetDir+"/"+"10-%s"+".conf", cniName)
	err := c.CreateFile(confFile, bytes, ConfFilePermission)
	if err != nil {
		return err
	}
	c.Files = append(c.Files, confFile)

	return nil
}

func (c *CNIConfig) GetCNIConfList(file string) (*libcni.NetworkConfigList, error) {
	return c.ParseCNIConfFromFile(file)
}

func (c *CNIConfig) LoadNetConfList(name string) (*libcni.NetworkConfigList, error) {
	return libcni.LoadConfList(c.NetDir, name)
}

func (c *CNIConfig) LoadConfFiles() error {
	files, err := c.ConfFiles(c.NetDir, []string{".conf", ".conflist"})
	if err != nil {
		return err
	}
	c.Files = files
	return nil
}

func (c *CNIConfig) GetConfFiles() ([]string, error) {
	return libcni.ConfFiles(c.NetDir, []string{".conf", ".conflist"})
}

func (c *CNIConfig) ParseCNIConfFromFile(file string) (*libcni.NetworkConfigList, error) {
	var err error
	var confList *libcni.NetworkConfigList
	if strings.HasSuffix(file, ".conflist") {
		confList, err = c.ConfListFromFile(file)
		if err != nil {
			return nil, fmt.Errorf("Error loading CNI config list file %s: %v", file, err)
		}
	} else {
		conf, err := c.ConfFromFile(file)
		if err != nil {
			return nil, fmt.Errorf("Error loading CNI config file %s: %v", file, err)
		}
		// Ensure the config has a "type" so we know what plugin to run.
		// Also catches the case where somebody put a conflist into a conf file.
		if conf.Network.Type == "" {
			return nil, fmt.Errorf("Error loading CNI config file %s: no 'type'; perhaps this is a .conflist?", file)
		}

		confList, err = c.ConfListFromConf(conf)
		if err != nil {
			return nil, fmt.Errorf("Error converting CNI config file %s to list: %v", file, err)
		}
	}
	if len(confList.Plugins) == 0 {
		return nil, fmt.Errorf("CNI config list %s has no networks", file)
	}
	return confList, nil
}

func (c *CNIConfig) ParseCNIConfFromBytes(bytes []byte) (*libcni.NetworkConfigList, error) {
	config := make(map[string]interface{})
	err := json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling config bytes: %v", err)
	}

	if _, ok := config["plugins"]; ok {
		return c.ConfListFromBytes(bytes)
	} else {
		return c.ConfListFromConfBytes(bytes)
	}
}
