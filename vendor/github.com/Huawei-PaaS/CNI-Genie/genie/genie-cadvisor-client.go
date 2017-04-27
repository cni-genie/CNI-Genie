// Copyright 2014 Google Inc. All Rights Reserved.
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


// Ref: package genie https://github.com/google/cadvisor/blob/master/client/client.go


// This is an implementation of a cAdvisor REST API in Go.
// To use it, create a client (replace the URL with your actual cAdvisor REST endpoint):
//   client, err := client.NewClient("http://192.168.59.103:8080/")
// Then, the client interface exposes go methods corresponding to the REST endpoints.

package genie

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"path"
	"strings"
	"github.com/google/cadvisor/info/v1"
	"github.com/golang/glog"
	. "github.com/Huawei-PaaS/CNI-Genie/utils"
	"time"
	"flag"
)

// Client represents the base URL for a cAdvisor client.
type Client struct {
	baseUrl    string
	httpClient *http.Client
}

// NewClient returns a new v1.3 client with the specified base URL.
func NewClient(url string) (*Client, error) {
	return newClient(url, http.DefaultClient)
}

// NewClientWithTimeout returns a new v1.3 client with the specified base URL and http client timeout.
func NewClientWithTimeout(url string, timeout time.Duration) (*Client, error) {
	return newClient(url, &http.Client{
		Timeout: timeout,
	})
}

func newClient(url string, client *http.Client) (*Client, error) {
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	return &Client{
		baseUrl:    fmt.Sprintf("%sapi/v1.3/", url),
		httpClient: client,
	}, nil
}

// MachineInfo returns the JSON machine information for this client.
// A non-nil error result indicates a problem with obtaining
// the JSON machine information data.
func (self *Client) MachineInfo() (minfo *v1.MachineInfo, err error) {
	u := self.machineInfoUrl()
	ret := new(v1.MachineInfo)
	if err = self.httpGetJsonData(ret, nil, u, "machine info"); err != nil {
		return
	}
	minfo = ret
	return
}

// ContainerInfo returns the JSON container information for the specified
// container and request.
func (self *Client) ContainerInfo(name string, query *v1.ContainerInfoRequest) (cinfo *v1.ContainerInfo, err error) {
	u := self.containerInfoUrl(name)
	ret := new(v1.ContainerInfo)
	if err = self.httpGetJsonData(ret, query, u, fmt.Sprintf("container info for %q", name)); err != nil {
		return
	}
	cinfo = ret
	return
}


// Returns the JSON container information for the specified
// Docker container and request.
func (self *Client) GetDockerContainers(query *v1.ContainerInfoRequest) (cinfo []ContainerStatsGenie, err error) {
	u := self.containerInfoUrl("/")
	//ret := make(map[string]ContainerInfoGenie)
	var containerInfoObj ContainerInfoGenie
	if err = self.httpGetJsonData(&containerInfoObj, query, u, "get all containers info"); err != nil {
		return nil, err
	}
	return containerInfoObj.Stats, nil
}

func (self *Client) machineInfoUrl() string {
	return self.baseUrl + path.Join("machine")
}

func (self *Client) containerInfoUrl(name string) string {
	return self.baseUrl + path.Join("containers", name)
}

func (self *Client) httpGetJsonData(data, postData interface{}, url, infoName string) error {
	var resp *http.Response
	var err error

	if postData != nil {
		data, marshalErr := json.Marshal(postData)
		if marshalErr != nil {
			return fmt.Errorf("unable to marshal data: %v", marshalErr)
		}
		resp, err = self.httpClient.Post(url, "application/json", bytes.NewBuffer(data))
	} else {
		resp, err = self.httpClient.Get(url)
	}
	if err != nil {
		return fmt.Errorf("unable to get %q from %q: %v", infoName, url, err)
	}
	if resp == nil {
		return fmt.Errorf("received empty response for %q from %q", infoName, url)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("unable to read all %q from %q: %v", infoName, url, err)
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("request %q failed with error: %q", url, strings.TrimSpace(string(body)))
	}

	if err = json.Unmarshal(body, data); err != nil {
		err = fmt.Errorf("unable to unmarshal %q (Body: %q) from %q with error: %v", infoName, string(body), url, err)
		return err
	}
	return nil
}

// This is getting overcomplicated
/**
This method returns string array of network solutions with ascending order of downlink usage.
eg: flannel=350, calico=250, weave=150

It returns {weave, calico, flannel}

*/
func computeNetworkUsage(cinfo []ContainerStatsGenie) ([]string) {
	//default ranks of usage
	//TODO (Karun): This is just a simple way of ranking. Needs improvement.
	//go with flannel as first preference, calico as second
	// weave as third
	m := make(map[string]int)
	m["flan"] 	= 1
	m["cali"] 	= 2
	m["weav"] 	= 3


	rx := make(map[string]uint64)
	//tx := make(map[string]uint64)

	for i, c := range cinfo {
		use := []InterfaceBandwidthUsage{}
		glog.V(6).Info("==== i=", i)
		for _, intf := range c.Network.Interfaces {
			var downlink uint64
			//var uplink uint64
			if _, ok := m[intf.Name[:4]]; ok {
				//fmt.Println("TxBytes=", intf.TxBytes)

				if oldrx, ok := rx[intf.Name]; ok {
					glog.V(6).Info("intfname=", intf.Name[:4])
					glog.V(6).Info("RxBytes=", intf.RxBytes)
					glog.V(6).Info("oldrx=", oldrx)
					downlink = math.Float64bits(math.Abs(math.Float64frombits(intf.RxBytes) - math.Float64frombits(oldrx)))
				}
				rx[intf.Name] = intf.RxBytes

				/*if oldtx, ok := tx[intf.Name]; ok {
					uplink = math.Float64bits(math.Abs(math.Float64frombits(intf.TxBytes) - math.Float64frombits(oldtx)))
					tx[intf.Name] = intf.TxBytes
				}*/
				use = append(use,InterfaceBandwidthUsage{IntName: intf.Name, DownLink: downlink})
				m[intf.Name[:4]] = int(downlink)
				//use = append(use,InterfaceBandwidthUsage{IntName: intf.Name, DownLink: downlink, UpLink: uplink})
			}
		}
		glog.V(6).Info("use==>", use)
	}
	glog.V(6).Info("m==>", m)
	//sort by values of map
	cns := SortedKeys(m)
	for i, c := range cns {
		if c == "weav" {
			cns[i] = "weave"
		} else if c == "flan" {
			cns[i] = "canal"
		} else if c == "cali" {
			cns[i] = "calico"
		}
	}
	fmt.Println("cns==>", cns)


	//fmt.Println("cns==>", cns)
	return cns
}

/**
Returns array of strings with network solutions in descending order of network usage.
input
	- cAdvisorURL : http://127.0.0.1:4194 or http://<nodeip>:4194
	- numStats : int (number of stats needed default 3)
 */
func GetCNSOrderByNetworkBandwith(cAdvisorURL string, numStats int) ([]string,error) {
	if cAdvisorURL == "" {
		return nil, fmt.Errorf("cAdvisorURL is empty. Should be eg: http://127.0.0.1:4194")
	}
	if numStats <= 0 {
		numStats = 3
	}

	staticClient, err := NewClient(cAdvisorURL)
	glog.V(6).Info("staticClient=", staticClient)
	if err != nil {
		glog.Errorf("tried to make client and got error %v", err)
		return nil, err
	}

	query := v1.ContainerInfoRequest{NumStats: numStats}

	cinfo, err := staticClient.GetDockerContainers(&query)
	if err != nil {
		glog.Errorf("got error retrieving event info: %v", err)
		return nil, err
	}
	res := computeNetworkUsage(cinfo)

	return res,nil

}

func staticContainersClientExample() {
	cns,err := GetCNSOrderByNetworkBandwith("http://127.0.0.1:4194/", 3)
	if err != nil {
		glog.Errorf("got error while fetching CNS: %v", err)
	}
	glog.V(6).Info("cns=", cns)

}

// demonstrates how to use event clients
func main() {
	flag.Parse()
	staticContainersClientExample()
}