# TeamCity Kubernetes Operator

Kubernetes operator to deploy and manage TeamCity servers. This repository contains a custom controller and custom resource definition (CRD) designed for the lifecycle (creation, upgrade, graceful shutdown) of a TeamCity server.

## Getting Started


These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

## Prerequisites
*If this is the first time, opening this project make sure that `go module integration` is enabled.* 

This setting is responsible for automatically downloading go mods specified in `go.mod` file. It's especially useful if lines in `go.mod` are highlighted with red colour.
![go_modules_setting.png](./docs/go_modules_setting.png)

Required go version:
```
go version go1.20.4
``` 
Required version of minikube. [Instructions for minikube setup](https://minikube.sigs.k8s.io/docs/start/)
```
minikube version: v1.30.1
```
To run locally/debug(configure *minikube* as your current context):
```
minikube start
kubectl config current-context
minikube
```

## Local webhook setup

In the root of the project run the following:

Setup dir for certificates(`certs` directory is already ignored in git, so it makes sense to use it):
```bash
mkdir -p certs
export CAROOT=$(pwd)/certs
```
`mkcert` is used to manage certificates and can be installed using:

**Mac**:
```bash
brew install mkcert
```
**Linux**:

```bash
curl -JLO "https://dl.filippo.io/mkcert/latest?for=linux/amd64"
chmod +x mkcert-v*-linux-amd64
sudo cp mkcert-v*-linux-amd64 /usr/local/bin/mkcert
```
Then, a new Root CA needs to be installed and certificates are generated. Note the name that is provided. If the developer, uses `Kind` cluster instead of `minikube` then `host.docker.internal` should be used instead(or as SAN). 

```bash
mkcert -install #this command will require sudo password
mkcert -cert-file=$CAROOT/tls.crt -key-file=$CAROOT/tls.key host.minikube.internal
```
This will place a few files in `certs` directory. 

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
## Run/Debug
Use run configuration `Run controller` in Run or Debug mode.

***This run configuration assumes minikube is running and context with name `minikube` exists.***

## Run/Debug with webhooks
Use run configuration `Run controller with webhooks` in Run or Debug mode.

***This run configuration assumes minikube is running and context with name `minikube` exists and steps in [Local webhook setup](#local-webhook-setup) are completed***


## Test

```shell
make test #finds all test files and runs ginkgo tests
```

## Deployment

```shell
make deploy #installs controller to the selected kube context
```


