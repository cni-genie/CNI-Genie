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
	"os"
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

func newClient(url string, client *http.Client) (*Client, error) {
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	return &Client{
		baseUrl:    fmt.Sprintf("%sapi/v1.3/", url),
		httpClient: client,
	}, nil
}

// Returns the JSON container information for the specified
// Docker container and request.
func GetDockerContainers(url string, query *v1.ContainerInfoRequest) (cinfo []ContainerStatsGenie, err error) {
	u := containerInfoUrl(url, "/")
	//ret := make(map[string]ContainerInfoGenie)
	var containerInfoObj ContainerInfoGenie
	if err = httpGetJsonData(&containerInfoObj, query, u, "get all containers info"); err != nil {
		return
	}
	cinfo = containerInfoObj.Stats
	return
}

func containerInfoUrl(baseUrl string, name string) string {
	return baseUrl + path.Join("containers", name)
}

func httpGetJsonData(data, postData interface{}, url, infoName string) error {
	var resp *http.Response
	var err error
	fmt.Fprintf(os.Stderr, "CAdvisor Client Inside httpGetJsonData() = %v\n", data)
	fmt.Fprintf(os.Stderr, "CAdvisor Client postData = %v\n", postData)
	/*if postData != nil {
		data, marshalErr := json.Marshal(postData)
		if marshalErr != nil {
			return fmt.Errorf("unable to marshal data: %v", marshalErr)
		}
		resp, err = client.httpClient.Post(url, "application/json", bytes.NewBuffer(data))
	} else {*/
		resp, err = http.DefaultClient.Get(url)
	//}
	fmt.Fprintf(os.Stderr, "CAdvisor Client resp = %v\n", resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CAdvisor Client err = %v\n", err)
		return fmt.Errorf("unable to get %q from %q: %v", infoName, url, err)
	}
	if resp == nil {
		return fmt.Errorf("received empty response for %q from %q", infoName, url)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	fmt.Fprintf(os.Stderr, "CAdvisor Client body = %v\n", string(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "CAdvisor Client err2 = %v\n", err)
		err = fmt.Errorf("unable to read all %q from %q: %v", infoName, url, err)
		return err
	}
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "CAdvisor Client Unmarshal resp.StatusCode = %v\n", resp.StatusCode)
		return fmt.Errorf("request %q failed with error: %q", url, strings.TrimSpace(string(body)))
	}

	if err = json.Unmarshal(body, data); err != nil {
		fmt.Fprintf(os.Stderr, "CAdvisor Client Unmarshal err = %v\n", err)
		err = fmt.Errorf("unable to unmarshal %q (Body: %q) from %q with error: %v", infoName, string(body), url, err)
		return err
	}
	fmt.Fprintf(os.Stderr, "CAdvisor Client data = %v\n", data)
	return nil
}

// This is getting overcomplicated
/**
This method returns string array of network solutions with ascending order of downlink usage.
eg: flannel=350, calico=250, weave=150

It returns {weave, calico, flannel}

*/
func computeNetworkUsage(cinfo []ContainerStatsGenie) (string) {
	//default ranks of usage
	//TODO (Karun): This is just a simple way of ranking. Needs improvement.
	//go with flannel as first preference, calico as second
	// weave as third
	m := make(map[string]int)
	m["weav"] 	= 1
	m["cali"] 	= 2
	m["flan"] 	= 3


	rx := make(map[string]uint64)
	//tx := make(map[string]uint64)

	for i, c := range cinfo {
		use := []InterfaceBandwidthUsage{}
		fmt.Fprintf(os.Stderr, "CAdvisor Client computeNetworkUsage i = %v\n", i)
		for _, intf := range c.Network.Interfaces {
			var downlink uint64
			//var uplink uint64
			if _, ok := m[intf.Name[:4]]; ok {
				//fmt.Println("TxBytes=", intf.TxBytes)

				if oldrx, ok := rx[intf.Name]; ok {
					fmt.Fprintf(os.Stderr, "CAdvisor Client computeNetworkUsage intfname = %v\n", intf.Name[:4])
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
		fmt.Fprintf(os.Stderr, "CAdvisor Client computeNetworkUsage use = %v\n", use)
		glog.V(6).Info("use==>", use)
	}
	fmt.Fprintf(os.Stderr, "CAdvisor Client computeNetworkUsage m = %v\n", m)
	glog.V(6).Info("m==>", m)
	//sort by values of map
	cns := SortedKeys(m)
	for i, c := range cns {
		if c == "weav" {
			cns[i] = "weave"
		} else if c == "flan" {
			cns[i] = "canal"
		} else if c == "cali" {
			// TODO (Karun): This is a bad fix.
			// Calico bin wasn't working correctly
			//cns[i] = "calico"
			cns[i] = "canal"
		}
	}
	fmt.Fprintf(os.Stderr, "CAdvisor Client computeNetworkUsage cns = %v\n", cns[0])
	fmt.Println("cns==>", cns)
	return cns[0]
}

/**
Returns array of strings with network solutions in descending order of network usage.
input
	- cAdvisorURL : http://127.0.0.1:4194 or http://<nodeip>:4194
	- numStats : int (number of stats needed default 3)
 */
func GetCNSOrderByNetworkBandwith(cAdvisorURL string, numStats int) (string,error) {
	if cAdvisorURL == "" {
		return "", fmt.Errorf("cAdvisorURL is empty. Should be eg: http://127.0.0.1:4194")
	}
	if numStats <= 0 {
		numStats = 3
	}

	/*staticClient, err := NewClient(cAdvisorURL)
	glog.V(6).Info("staticClient=", staticClient)
	if err != nil {
		glog.Errorf("tried to make client and got error %v", err)
		return nil, err
	}*/

	//query := v1.ContainerInfoRequest{NumStats: numStats}
	/*	query := v1.DefaultContainerInfoRequest()

		fmt.Println("query==>", query)
*/
	cinfo, _ := GetDockerContainers(fmt.Sprintf("%s/api/v1.3/", cAdvisorURL), nil)
	/*if err != nil {
		glog.Errorf("got error retrieving event info: %v", err)
		fmt.Errorf("****got error retrieving event info: %v", err)
		return nil, err
	}*/
	fmt.Fprintf(os.Stderr, "CAdvisor Client cinfo is = %v\n", cinfo)
	//res := computeNetworkUsage(cinfo)

	//res := []string{"canal", "weave", "test"}
	res := "canal"
	fmt.Fprintf(os.Stderr, "CAdvisor Client response = %v\n", res)
	return res,nil

}
/*
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
*/
