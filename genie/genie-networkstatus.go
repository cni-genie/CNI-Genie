package genie

import (
	"encoding/json"
	"fmt"
	"github.com/cni-genie/CNI-Genie/networkcrd"
	"github.com/cni-genie/CNI-Genie/utils"
	"github.com/containernetworking/cni/pkg/types/current"
	"os"
	"strconv"
)

func setGenieStatus(result current.Result, name, ifName string, currStatus interface{}) interface{} {
	multiIPPreferences := &utils.MultiIPPreferences{}
	var ok bool
	if currStatus == nil {
		multiIPPreferences.MultiEntry = 1
		multiIPPreferences.Ips = make(map[string]utils.IPAddressPreferences)
	} else {
		multiIPPreferences, ok = currStatus.(*utils.MultiIPPreferences)
		if !ok {
			fmt.Fprintf(os.Stderr, "CNI Genie unable to assert multiIPPreferences\n")
			return nil
		}
		multiIPPreferences.MultiEntry = multiIPPreferences.MultiEntry + 1
	}

	if len(result.IPs) == 0 {
		fmt.Fprintf(os.Stderr, "CNI Genie no ip in result\n")
		return nil
	}
	multiIPPreferences.Ips["ip"+strconv.Itoa(int(multiIPPreferences.MultiEntry))] = utils.IPAddressPreferences{
		Ip:        result.IPs[0].Address.IP.String(),
		Interface: ifName,
	}
	return interface{}(multiIPPreferences)
}

func setNetAttachStatus(result current.Result, name, ifName string, currStatus interface{}) interface{} {
	netAttachStatus := &[]networkcrd.NetworkStatus{}
	status := networkcrd.NetworkStatus{}
	var ok bool
	if currStatus != nil {
		netAttachStatus, ok = currStatus.(*[]networkcrd.NetworkStatus)
		if !ok {
			fmt.Fprintf(os.Stderr, "CNI Genie unable to assert network attachment status\n")
			return nil
		}
	} else {
		status.Default = true
	}

	for _, intf := range result.Interfaces {
		if intf.Sandbox != "" {
			status.Mac = intf.Mac
		}
	}

	for _, ip := range result.IPs {
		if ip.Version == "4" && ip.Address.IP.To4() != nil {
			status.IPs = append(status.IPs, ip.Address.IP.String())
		} else if ip.Version == "6" && ip.Address.IP.To16() != nil {
			status.IPs = append(status.IPs, ip.Address.IP.String())
		}
	}

	status.Name = name
	status.Interface = ifName
	status.DNS = result.DNS

	*netAttachStatus = append(*netAttachStatus, status)

	return interface{}(netAttachStatus)
}

func getStatusBytes(status interface{}) []byte {
	var bytes []byte
	var err error
	if nwStatus, ok := status.(*utils.MultiIPPreferences); ok {
		bytes, err = json.Marshal(*nwStatus)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie error while marshalling status: %v\n", err)
		}
	} else if nwStatus, ok := status.(*[]networkcrd.NetworkStatus); ok {
		bytes, err = json.MarshalIndent(nwStatus, "", " ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "CNI Genie error while marshalling network attachment status: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "CNI Genie unable to extract status information\n")
	}

	return bytes
}
