# tch/teamcity-operator



## Getting Started

Download links:

SSH clone URL: ssh://git@git.jetbrains.team/tch/teamcity-operator.git

HTTPS clone URL: https://git.jetbrains.team/tch/teamcity-operator.git



These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

## Prerequisites

```
go version go1.20.4
``` 
```
minikube version: v1.30.1
```
To run locally/debug(configure *minikube* as your current context):
```
kubectl config current-context
minikube
```

## Project layout

```
├── api
│   └── v1alpha1 #structs that represent crds. run make generate to generate yaml crds into config/crd/bases folders
├── bin 
│   └── k8s
│       └── 1.26.0-darwin-arm64 #results of make build
├── cmd #contains main.go. operator's entrypoint 
├── config #contains yaml resources for the oeprator to function
│   ├── crd
│   │   ├── bases
│   │   └── patches
│   ├── default
│   ├── manager
│   ├── manifests
│   ├── prometheus #prometheus resources. normally should not be touched
│   ├── rbac #required rbac resources
│   ├── samples #sample yaml resource we can use for testing
│   └── scorecard #olm. normally should not be touched
│       ├── bases
│       └── patches
├── hack #contains boilerplate. normally should not be touched
└── internal #operator's internals
    ├── controller #contains reconciler loop
    ├── metadata #contains functions for building labels and annotations
    └── resource #contains builders for each resource 

25 directories

```
## Run

```shell
make run
```

## Debug
Use run configuration `Run controller` in Run or Debug mode.

***This run configuration assumes minikube is running and context with name `minikube` exists.***

## Test

```shell
make test #finds all test files and runs ginkgo tests
```

## Deployment

```shell
make deploy #installs controller to the selected kube context
```

## Resources

Add links to external resources for this project, such as CI server, bug tracker, etc.

