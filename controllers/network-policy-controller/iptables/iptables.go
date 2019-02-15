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
package iptable

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/cni-genie/CNI-Genie/utils"
	"github.com/coreos/go-iptables/iptables"
	"github.com/golang/glog"
	"strconv"
	"strings"
)

const (
	GenieBaseNPCChain  = "Genie-NPC-Base"
	FilterTable        = "filter"
	ForwardChain       = "FORWARD"
	GeniePolicyPrefix  = "GnPlc-"
	GenieNetworkPrefix = "GnNtk-"
	// defaultRuleCountForNetworkChain specifies the number of rules present in a
	// network chain when no policy chain rules are present in the network chain
	defaultRuleCountForNetworkChain = 3
)

type IpTables struct {
	*iptables.IPTables
}

func ExitStatus(err error) int {
	return err.(*iptables.Error).ExitStatus()
}

func CreateIptableChainName(prefix, suffix string) string {
	m := md5.Sum([]byte(suffix))
	return (prefix + fmt.Sprintf("%x", m))[:26]
}

func CreatePolicyChainName(name, namespace, args string) string {
	policyChain := CreateIptableChainName("", name+namespace)[:10]
	nwPolicyChainName := CreateIptableChainName(GeniePolicyPrefix+policyChain+"-", args)
	return nwPolicyChainName
}

// CreateBaseChain creates the base policy chain
func CreateBaseChain() (IpTables, error) {
	var iptable IpTables
	var err error
	iptable.IPTables, err = iptables.New()
	if err != nil {
		return IpTables{}, fmt.Errorf("Iptables command executer intialization failed: %v", err)
	}

	err = iptable.ClearChain(FilterTable, GenieBaseNPCChain)
	if err != nil {
		return IpTables{}, err
	}

	rulespec := []string{"-j", GenieBaseNPCChain}
	exists, err := iptable.Exists(FilterTable, ForwardChain, rulespec...)
	if err != nil {
		return IpTables{}, fmt.Errorf("Error while checking for Genie base rule in FORWARD chain: %v", err)
	}
	if !exists {
		err = iptable.Insert(FilterTable, ForwardChain, 1, rulespec...)
		if err != nil {
			return IpTables{}, fmt.Errorf("Error inserting a rule for Genie npc: %v", err)
		}
	}

	return iptable, nil
}

// ExistsChain chaecks for existence of a chain in filter table
func (i *IpTables) ExistsChain(chain string) bool {
	_, err := i.List(FilterTable, chain)
	if err != nil {
		if err.(*iptables.Error).ExitStatus() == 1 {
			return false
		} else {
			return true
		}
	}

	return true
}

// AddPolicyChain adds a policy specific chain
func (i *IpTables) AddPolicyChain(name, namespace, nwSelector string) (string, error) {
	nwPolicyChainName := CreatePolicyChainName(name, namespace, nwSelector)

	err := i.ClearChain(FilterTable, nwPolicyChainName)
	if err != nil {
		return "", err
	}

	return nwPolicyChainName, nil
}

// AddNetworkChain adds a logical network specific chain
func (i *IpTables) AddNetworkChain(ln *utils.LogicalNetwork) (string, error) {
	lnChain := CreateIptableChainName(GenieNetworkPrefix, ln.Name+ln.Namespace)

	err := i.ClearChain(FilterTable, lnChain)
	if err != nil {
		return "", err
	}

	args := []string{"-s", ln.Spec.SubSubnet, "-d", ln.Spec.SubSubnet, "-j", "ACCEPT"}
	err = i.AppendUnique(FilterTable, lnChain, args...)
	if err != nil {
		return "", err
	}

	args = []string{"-j", "REJECT"}
	err = i.AppendUnique(FilterTable, lnChain, args...)
	if err != nil {
		return "", err
	}

	args = []string{"-d", ln.Spec.SubSubnet, "-j", lnChain}
	exists, err := i.Exists(FilterTable, GenieBaseNPCChain, args...)
	if err != nil {
		return "", err
	}
	if !exists {
		err := i.AppendUnique(FilterTable, GenieBaseNPCChain, args...)
		if err != nil {
			return "", err
		}
	}

	args = []string{"-s", ln.Spec.SubSubnet, "-j", lnChain}
	exists, err = i.Exists(FilterTable, GenieBaseNPCChain, args...)
	if err != nil {
		return "", err
	}
	if !exists {
		err := i.AppendUnique(FilterTable, GenieBaseNPCChain, args...)
		if err != nil {
			return "", err
		}
	}

	return lnChain, nil
}

// DeleteNetworkChain deletes the entry of the network chain from
// the base npc chain and then deletes the chain
func (i *IpTables) DeleteNetworkChain(chain string) error {
	rules, err := i.List(FilterTable, GenieBaseNPCChain)
	if err != nil {
		return fmt.Errorf("Failed to list rules for Genie base chain: %v", err)
	}

	var pos, cnt int
	for _, rule := range rules {
		if strings.Contains(rule, chain) && cnt < 2 {
			cnt++
			err = i.Delete(FilterTable, GenieBaseNPCChain, strconv.Itoa(pos))
			if err != nil {
				glog.Errorf("Error deleting rule for network chain (%s) from Genie base chain: %v", chain, err)
			} else {
				pos--
			}
		}
		if cnt == 2 {
			break
		}
		pos++
	}

	err = i.DeleteIptableChain(FilterTable, chain)
	if err != nil {
		return err
	}

	return nil
}

// InsertRule inserts a if the rule does not already exist
func (i *IpTables) InsertRule(chain string, pos int, args []string) error {
	exists, err := i.Exists(FilterTable, chain, args...)
	if err != nil {
		return err
	}
	if !exists {
		err := i.Insert(FilterTable, chain, pos, args...)
		if err != nil && err.(*iptables.Error).ExitStatus() != 1 {
			return err
		}
	}

	return nil
}

// DeleteNetworkChainRule deletes a rule from network chain
// and also deletes the chain if the rule was the last one specifying policy
func (i *IpTables) DeleteNetworkChainRule(nwChain string, rules []string) ([]string, error) {
	nwChainRules, err := i.List(FilterTable, nwChain)
	if err != nil {
		return nil, fmt.Errorf("Error listing rules for network chain (%s): %v", nwChain, err)
	}
	glog.V(6).Infof("Network chain rules for network chain %s: %v", nwChain, nwChainRules)
	policyRules := make(map[string]map[string]int)
	for pos, rule := range nwChainRules {
		if strings.Contains(rule, GeniePolicyPrefix) {
			rule = rule[strings.LastIndex(rule, " ")+1:]
			lastindex := strings.LastIndex(rule, "-")
			if policyRules[rule[:lastindex+1]] == nil {
				policyRules[rule[:lastindex+1]] = make(map[string]int)
			}
			policyRules[rule[:lastindex+1]][rule[lastindex+1:]] = pos
		}
	}
	glog.V(6).Infof("Policy rules map with policy rule positions: %v", policyRules)
	rulesDeleted := make([]string, 0)
	var errmsg string
	cnt := 0
	lastPos := 0
	for _, r := range rules {
		glog.V(4).Infof("Policy rule to be removed from network chain %s is %s", nwChain, r)
		policy := r[:strings.LastIndex(r, "-")+1]
		if policyRules[policy] != nil {
			selector := r[strings.LastIndex(r, "-")+1:]
			if policyRules[policy][selector] != 0 {
				delPos := policyRules[policy][selector]
				if delPos > lastPos {
					delPos = delPos - cnt
				}
				glog.V(6).Infof("Deleting rule (%s) from network chain (%s): position: %s", r, nwChain, delPos)
				err = i.Delete(FilterTable, nwChain, strconv.Itoa(delPos))
				if err != nil {
					errmsg = errmsg + fmt.Sprintf("Error deleting rule (%s) from network chain (%s): %v;\t", policy+selector, nwChain, err)
				} else {
					cnt++
					lastPos = delPos
					rulesDeleted = append(rulesDeleted, policy+selector)
				}
			} else if r == policy {
				for s, delPos := range policyRules[policy] {
					if delPos > lastPos {
						delPos = delPos - cnt
					}
					glog.V(6).Infof("Deleting rule (%s) from network chain (%s): position: %s", r, nwChain, delPos)
					err = i.Delete(FilterTable, nwChain, strconv.Itoa(delPos))
					if err != nil {
						errmsg = errmsg + fmt.Sprintf("Error deleting rule (%s) from network chain (%s): %v;\t", policy+s, nwChain, err)
					} else {
						cnt++
						lastPos = delPos
						rulesDeleted = append(rulesDeleted, policy+s)
					}
				}
			}
		}
	}

	if cnt >= len(nwChainRules)-defaultRuleCountForNetworkChain {
		glog.V(4).Infof("Deleing network chain %s", nwChain)
		err = i.DeleteNetworkChain(nwChain)
		if err != nil {
			errmsg = errmsg + fmt.Sprintf("Error deleting network chain (%s): %v;\t", nwChain, err)
		}
	}

	if errmsg == "" {
		return rulesDeleted, nil
	} else {
		return rulesDeleted, errors.New(errmsg)
	}

}

// DeleteIptableChain deletes a chain in the given table
func (i *IpTables) DeleteIptableChain(table, chain string) error {
	err := i.ClearChain(table, chain)
	if err != nil {
		return fmt.Errorf("Error flushing iptable chain (%s) before deleting it: %v", chain, err)
	}
	err = i.DeleteChain(table, chain)
	if err != nil {
		return fmt.Errorf("Error deleting iptable chain (%s): %v", chain, err)
	}

	return nil
}
