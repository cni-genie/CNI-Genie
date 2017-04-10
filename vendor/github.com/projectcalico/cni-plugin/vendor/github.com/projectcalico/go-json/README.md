[![Build Status](https://semaphoreci.com/api/v1/calico/go-json/branches/calico/badge.svg)](https://semaphoreci.com/calico/go-json)

# go-json

A "fork" of the golang encoding/json module.  Rather than forking the entire
golang-go repo, we've taken a snapshot of the encoding/json sub directory of:

https://github.com/golang/go
commit 75055de84ab7ad0f36b4c93e5c851ea55b297c95

To active branch for calico is `calico`
The branch `master` is fixed to the above snapshot.

The calico branch contains a suggested enhancement to disallow unknown fields.  The original fix was written by
Michael Spiegel <michael.m.spiegel@gmail.com>, and can be found in the following submission:
-  https://go-review.googlesource.com/#/c/27231/

The calico branch also contains additional tweaks to:  
-  change the error messages to sound less code-oriented (remove the word json, since we also use this for yaml parsing, don't use the word marshal/unmarshal)
-  Add additional info into the field error to indicate the parsed field value
