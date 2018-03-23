/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// only allow logical networks objects to be created only when all input validations are passed
func admit(data []byte) *v1alpha1.AdmissionReviewStatus {
	ar := v1alpha1.AdmissionReview{}
	if err := json.Unmarshal(data, &ar); err != nil {
		glog.Error(err)
		return nil
	}
	// The externalAdmissionHookConfiguration registered via selfRegistration
	// asks the kube-apiserver only sends admission request regarding logical networks.
	logicalNwResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "logicalnetworks"}
	if ar.Spec.Resource != logicalNwResource {
		glog.Errorf("expect resource to be %s", logicalNwResource)
		return nil
	}

	reviewStatus := v1alpha1.AdmissionReviewStatus{}
	/* Validation for logical network to be done here*/
	reviewStatus.Allowed = true
	return &reviewStatus
}

// Will be called whenever user create logical network object
func serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	reviewStatus := admit(body)
	ar := v1alpha1.AdmissionReview{
		Status: *reviewStatus,
	}

	resp, err := json.Marshal(ar)
	if err != nil {
		glog.Error(err)
	}
	if _, err := w.Write(resp); err != nil {
		glog.Error(err)
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/", serve)
	clientset := getClient()
	server := &http.Server{
		Addr:      ":8000",
		TLSConfig: configTLS(clientset),
	}
	go selfRegistration(clientset, caCert)
	server.ListenAndServeTLS("", "")
}
