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

package genie

/**
This class enables cni-genie to check network usage interface wise on a node.
In other words, it primarily computes network usage for each of the CNS on a
given node i.e. what is the network usage of weave, flannel etc.

It returns CNS that has least load on it. So that, cni-genie can configure
networking on the CNS with least load.
*/

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/cni-genie/CNI-Genie/utils"
	"github.com/google/cadvisor/info/v1"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

const (
	// DefaultCAdvisorPath specifies the default address at which CAdvisor is running
	DefaultCAdvisorPath = "http://127.0.0.1:4194"
)

type Cadvisor interface {
	Get(url string) (*http.Response, error)
	Post(url string, contentType string, body io.Reader) (*http.Response, error)
}

// Client represents the base URL for a cAdvisor client.
type CadClient struct {
	httpClient *http.Client
}

func getCadClient() *CadClient {
	return &CadClient{httpClient: http.DefaultClient}
}

// Returns the JSON container information for the specified
// Docker container and request.
func (gc *GenieController) GetDockerContainers(url string, query *v1.ContainerInfoRequest) (cinfo []ContainerStatsGenie, err error) {
	u := containerInfoUrl(url, "/")
	var containerInfoObj ContainerInfoGenie
	if err = httpGetJsonData(&containerInfoObj, query, u, "get all containers info", gc.Cad); err != nil {
		return
	}
	cinfo = containerInfoObj.Stats
	return
}

func containerInfoUrl(baseUrl string, name string) string {
	return baseUrl + path.Join("containers", name)
}

func httpGetJsonData(data, postData interface{}, url, infoName string, c Cadvisor) error {
	var resp *http.Response
	var err error
	fmt.Fprintf(os.Stderr, "CAdvisor Client Inside httpGetJsonData() = %v\n", data)
	fmt.Fprintf(os.Stderr, "CAdvisor Client postData = %v\n", postData)
	if postData != nil {
		data, marshalErr := json.Marshal(postData)
		if marshalErr != nil {
			return fmt.Errorf("unable to marshal data: %v", marshalErr)
		}
		resp, err = c.Post(url, "application/json", bytes.NewBuffer(data))
	} else {
		resp, err = c.Get(url)
	}
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

// This is getting overcomplicated. In future needs to be re-written.
/**
This method returns string array of network solutions with ascending order of downlink usage.
eg: flannel=350, calico=250, weave=150

It returns {weave, calico, flannel}

*/
func computeNetworkUsage(cinfo []ContainerStatsGenie) string {
	//default ranks of usage
	//TODO (Karun): This is just a simple way of ranking. Needs improvement.
	//go with flannel as first preference, calico as second
	// weave as third
	m := make(map[string]int)
	m["flan"] = 0
	m["cali"] = 0
	m["weav"] = 0
	var downlink int

	rx := make(map[string]uint64)

	//TODO (Karun): Need to rethink on the logic. This is not an accurate measure.
	for i, c := range cinfo {
		fmt.Fprintf(os.Stderr, "CAdvisor Client computeNetworkUsage i = %v\n", i)
		for _, intf := range c.Network.Interfaces {
			if _, ok := m[intf.Name[:4]]; ok {
				if oldrx, ok := rx[intf.Name]; ok {
					fmt.Fprintf(os.Stderr, "CAdvisor Client computeNetworkUsage intfname = %v\n", intf.Name[:4])
					fmt.Fprintf(os.Stderr, "CAdvisor Client computeNetworkUsage intf.RxBytes = %v\n", intf.RxBytes)
					fmt.Fprintf(os.Stderr, "CAdvisor Client computeNetworkUsage oldrx = %v\n", oldrx)
					downlink = int(intf.RxBytes - oldrx)
				}
				rx[intf.Name] = intf.RxBytes
				m[intf.Name[:4]] = downlink
			}
		}
	}
	fmt.Fprintf(os.Stderr, "CAdvisor Client computeNetworkUsage m = %v\n", m)
	//sort by values of map
	cns := SortedKeys(m)
	for i, c := range cns {
		if c == "weav" {
			cns[i] = "weave"
		} else if c == "flan" {
			cns[i] = "flannel"
		} else if c == "cali" {
			// TODO (Karun): This is a bad fix.
			// Calico bin wasn't working correctly
			//cns[i] = "calico"
			cns[i] = "calico"
		}
	}
	fmt.Fprintf(os.Stderr, "CAdvisor Client computeNetworkUsage cns = %v\n", cns[0])
	return cns[0]
}

/**
Returns network solution that has least load
	- conf : Netconf info having genie configurations
	- numStats : int (number of stats needed default 3)
*/
func (gc *GenieController) GetCNSOrderByNetworkBandwith(conf *GenieConf) (string, error) {
	cAdvisorURL := getCAdvisorAddr(conf)

	cinfo, err := gc.GetDockerContainers(fmt.Sprintf("%s/api/v1.3/", cAdvisorURL), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CAdvisor Client cinfo err = %v\n", err)
		return "", err
	}
	fmt.Fprintf(os.Stderr, "CAdvisor Client cinfo is = %v\n", cinfo)
	res := computeNetworkUsage(cinfo)
	fmt.Fprintf(os.Stderr, "CAdvisor Client response = %v\n", res)
	return res, nil
}

/**
Returns cAdvisor Address to collect network usage parameters
	- conf : Netconf info having genie configurations
*/
func getCAdvisorAddr(conf *GenieConf) string {
	conf.CAdvisorAddr = strings.TrimSpace(conf.CAdvisorAddr)
	if conf.CAdvisorAddr == "" {
		return DefaultCAdvisorPath
	}
	return conf.CAdvisorAddr
}

func (c *CadClient) Get(url string) (*http.Response, error) {
	return c.httpClient.Get(url)
}

func (c *CadClient) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	return c.httpClient.Post(url, contentType, body)
}
