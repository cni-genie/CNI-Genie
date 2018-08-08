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
package ipset

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

type IpSet struct {
	path string
}

type output struct {
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func getPath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	return path, nil
}

func New() (*IpSet, error) {
	path, err := getPath("ipset")
	if err != nil {
		return nil, err
	}
	return &IpSet{path: path}, nil
}

func (i *IpSet) Create(set, setType, family string, timeout uint) error {
	if setType == "" {
		setType = "hash:ip"
	}
	if family == "" {
		family = "inet"
	}
	args := []string{"create", set, setType, "family", family, "timeout", strconv.Itoa(int(timeout))}
	_, err := i.run(args...)
	return err
}

func (i *IpSet) Add(set, entry string) error {
	args := []string{"add", set, entry}
	_, err := i.run(args...)
	return err
}

func (i *IpSet) AddExist(set, entry string) error {
	args := []string{"add", set, entry, "-exist"}
	_, err := i.run(args...)
	return err
}

func (i *IpSet) Del(set, entry string) error {
	args := []string{"del", set, entry}
	_, err := i.run(args...)
	return err
}

func (i *IpSet) List(set string) ([]string, error) {
	args := []string{"list", set}
	out, err := i.run(args...)
	if err != nil {
		return nil, err
	}
	entries := strings.Split(out.stdout.String()[strings.Index(out.stdout.String(), "Members:")+len("Members:")+1:], "\n")
	return entries, nil
}

func (i *IpSet) ListSets() ([]string, error) {
	args := []string{"list", "-name"}
	out, err := i.run(args...)
	if err != nil {
		return nil, err
	}
	return strings.Split(out.stdout.String(), "\n"), nil
}

func (i *IpSet) ListMatchedSets(match string) ([]string, error) {
	sets, err := i.ListSets()
	if err != nil {
		return nil, err
	}

	var matchedSets []string
	for _, set := range sets {
		if strings.HasPrefix(set, match) {
			matchedSets = append(matchedSets, set)
		}
	}

	return matchedSets, nil
}

func (i *IpSet) Exists(set string) bool {
	out, err := i.run([]string{"list", set}...)
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			return !(1 == e.Sys().(syscall.WaitStatus).ExitStatus() &&
				strings.Contains(out.stderr.String(), "The set with the given name does not exist"))
		}
	}
	return true
}

func (i *IpSet) Flush(set string) error {
	args := []string{"flush", set}
	_, err := i.run(args...)
	return err
}

func (i *IpSet) Destroy(set string) error {
	args := []string{"destroy", set}
	_, err := i.run(args...)
	return err
}

func (i *IpSet) run(args ...string) (*output, error) {
	var out output
	cmd := exec.Cmd{
		Path:   i.path,
		Args:   append([]string{i.path}, args...),
		Stdout: &out.stdout,
		Stderr: &out.stderr,
	}
	err := cmd.Run()
	return &out, err
}
