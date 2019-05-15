package interfaces

import (
	"context"
	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
)

type InvokeExec interface {
	InvokeExecAdd(list *libcni.NetworkConfigList, rt *libcni.RuntimeConf) (types.Result, error)
	InvokeExecDel(list *libcni.NetworkConfigList, rt *libcni.RuntimeConf) error
}

type Invoke struct {
	Path []string
}

func (i *Invoke) InvokeExecAdd(config *libcni.NetworkConfigList, rtConf *libcni.RuntimeConf) (types.Result, error) {
	cniConfig := libcni.CNIConfig{Path: i.Path}
	ctx := context.TODO()
	return cniConfig.AddNetworkList(ctx, config, rtConf)
}

func (i *Invoke) InvokeExecDel(config *libcni.NetworkConfigList, rtConf *libcni.RuntimeConf) error {
	cniConfig := libcni.CNIConfig{Path: i.Path}
	ctx := context.TODO()
	return cniConfig.DelNetworkList(ctx, config, rtConf)
}
