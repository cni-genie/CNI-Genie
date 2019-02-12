package interfaces

import "github.com/containernetworking/cni/libcni"

type CNI interface {
	ConfListFromFile(file string) (*libcni.NetworkConfigList, error)
	ConfListFromBytes(bytes []byte) (*libcni.NetworkConfigList, error)
	ConfFromFile(file string) (*libcni.NetworkConfig, error)
	ConfFromBytes(bytes []byte) (*libcni.NetworkConfig, error)
	ConfListFromConf(conf *libcni.NetworkConfig) (*libcni.NetworkConfigList, error)
	ConfListFromConfBytes(confBytes []byte) (*libcni.NetworkConfigList, error)
	ConfFiles(dir string, ext []string) ([]string, error)
}

type Cni struct{}

func (_ *Cni) ConfListFromFile(file string) (*libcni.NetworkConfigList, error) {
	return libcni.ConfListFromFile(file)
}

func (_ *Cni) ConfListFromBytes(bytes []byte) (*libcni.NetworkConfigList, error) {
	return libcni.ConfListFromBytes(bytes)
}

func (_ *Cni) ConfFromFile(file string) (*libcni.NetworkConfig, error) {
	return libcni.ConfFromFile(file)
}

func (_ *Cni) ConfFromBytes(bytes []byte) (*libcni.NetworkConfig, error) {
	return libcni.ConfFromBytes(bytes)
}

func (_ *Cni) ConfListFromConf(conf *libcni.NetworkConfig) (*libcni.NetworkConfigList, error) {
	return libcni.ConfListFromConf(conf)
}

func (_ *Cni) ConfFiles(dir string, ext []string) ([]string, error) {
	return libcni.ConfFiles(dir, ext)
}

func (_ *Cni) ConfListFromConfBytes(confBytes []byte) (*libcni.NetworkConfigList, error) {
	conf, err := libcni.ConfFromBytes(confBytes)
	if err != nil {
		return nil, err
	}
	confList, err := libcni.ConfListFromConf(conf)
	if err != nil {
		return nil, err
	}
	return confList, nil
}
