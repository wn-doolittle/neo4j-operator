# Neo4J Operator

[![License](https://img.shields.io/github/license/lukasz-antoniak/neo4j-operator.svg)](https://raw.githubusercontent.com/lukasz-antoniak/neo4j-operator/master/LICENSE) [![Docker Hub](https://img.shields.io/docker/pulls/lantonia/neo4j-operator.svg)](https://hub.docker.com/r/lantonia/neo4j-operator)

#### Project status: alpha

The project is in alpha phase. While no breaking API changes are currently planned, we reserve the right to address bugs and change the API before the project is declared stable.

## Table of Contents

* [Overview](#overview)
* [Usage](#usage)    
    * [Install the Operator](#install-the-operator)
    * [Deploy Sample Neo4J Cluster](#deploy-sample-neo4j-cluster)
    * [Uninstall the Operator](#uninstall-the-operator)
* [Neo4J Cluster Configuration](#neo4j-cluster-configuration)
    * [Basic](#basic)
    * [SSL Certificates](#ssl-certificates)
    * [Scheduled Backups](#scheduled-backups)
* [Development](#development)
    * [Run the Operator Locally](#run-the-operator-locally)
    * [Build the Operator Image](#build-the-operator-image)
    * [Direct Access to Neo4J Cluster](#direct-access-to-neo4j-cluster)

## Overview

The operator itself has been built with the [Operator framework](https://github.com/operator-framework/operator-sdk) and runs official Neo4J [Docker image](https://hub.docker.com/_/neo4j/).

## Usage

### Install the Operator

    $ kubectl apply -f deploy/operator.yaml

Operator requires elevated privileges in order to watch for the custom resource updates. On Google Kubernetes Engine, the following command must be run before continuing with installation process. Replace user ID with your own credentials.

    $ cat <<EOF | kubectl apply -f -
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: your-google-id@gmail.com-cluster-admin-binding
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: cluster-admin
    subjects:
    - kind: User
      name: "your-google-id@gmail.com"
    EOF

## Deploy Sample Neo4J Cluster

Create file `neo4j.yaml` with the following content to provision eight node cluster.

    apiVersion: database.neo4j.org/v1alpha1
    kind: Neo4jCluster
    metadata:
      name: example1
    spec:
      image-version: 3.5.4-enterprise
      admin-password: TmVvNEojUGFzc3cwcmQxMjMh
      core-replicas: 3
      read-replica-replicas: 5
      resources:
        requests:
          cpu: 1000m
          memory: 2048Mi
        limits:
          cpu: 1000m
          memory: 2048Mi
      persistent-storage:
        size: 1Gi

Deploy Neo4J cluster.

    $ kubectl apply -f neo4j.yaml

Query cluster status to review ready servers and the current leader.

    $ kubectl get neo4jclusters.database.neo4j.org
    NAME       CORE SERVERS   READ REPLICAS   LEADER                                      BOLT URL
    example1   3/3            5/5             neo4j-core-example1-1.neo4j-core-example1   bolt+routing://neo4j-core-example1:7687

## Uninstall the Operator

Neo4J pods, volumes and services managed by the operator will not be deleted even if the operator is uninstalled.

    $ kubectl delete -f deploy/operator.yaml

## Neo4J Cluster Configuration

### Basic

Below table presents basic configuration parameters.

| Parameter name          | Parameter type | Description                                                                             | Example                  |
|-------------------------|----------------|-----------------------------------------------------------------------------------------|--------------------------|
| `image-version`         | string         | Version of official Neo4J Docker image. Neo4J cluster requires _enterprise_ image type. | 3.5.4-enterprise         |
| `admin-password`        | string         | Base64 encoded admin password.                                                          | TmVvNEojUGFzc3cwcmQxMjMh |
| `core-replicas`         | number         | Number of core replica servers.                                                         | 3                        |
| `core-args`             | map            | Map of additional arguments to be passed to core servers.                               |                          |
| `read-replica-replicas` | number         | Number of read replicas.                                                                | 5                        |
| `read-replica-args`     | map            | Map of additional environment variables to be passed to read replicas.                  |                          |
| `resources`             |                | Standard Kubernetes definition of requested CPU and memory resources.                   | See below                |
| `persistent-storage`    |                | Specifies details of persistent storage attached to core servers and read replicas.     | See below                |

Complete example of eight node cluster definition.

    apiVersion: database.neo4j.org/v1alpha1
    kind: Neo4jCluster
    metadata:
      name: example1
    spec:
      image-version: 3.5.4-enterprise
      admin-password: TmVvNEojUGFzc3cwcmQxMjMh
      core-replicas: 3
      read-replica-replicas: 5
      resources:
        requests:
          cpu: 1000m
          memory: 2048Mi
        limits:
          cpu: 1000m
          memory: 2048Mi
      persistent-storage:
        size: 1Gi

### SSL Certificates

Users may choose to leverage custom SSL certificates for encrypted communication. Paste content of private key and public certificate to Neo4J cluster YAML file.

    ssl:
      key: |
        -----BEGIN PRIVATE KEY-----
        MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDI05vZ7So8Ly6D
        ... (truncated) ...
        U2RRbKeEcDhcABzPF0bJFEPR
        -----END PRIVATE KEY-----
      certificate: |
        -----BEGIN CERTIFICATE-----
        MIIDkzCCAnugAwIBAgIJAJMfoyRqXIZMMA0GCSqGSIb3DQEBBQUAMGAxCzAJBgNV
        ... (truncated) ...
        DQnQ+OGUMw==
        -----END CERTIFICATE-----

### Scheduled Backups

Operator can manage automatic backups triggered as Kubernetes cron jobs. Users have to define schedule, storage size and resource limits for pod executing `neo4j-admin backup` command.

    backup:
      schedule: "*/5 * * * *"
      size: 2Gi
        resources:
          requests:
            cpu: 200m
            memory: 1024Mi
          limits:
            cpu: 200m
            memory: 1024Mi

## Development

### Requirements

* `kubectl` >= 1.12
* `go` >= 1.13
* `kubebuilder` >= 2.3.1
* `kustomize` >= 3.8.1 -- cli (`brew install kustomize`)

### Run the Operator Locally

You can run the operator locally to help with development, testing, and debugging tasks.

The following command will run the operator locally with the default Kubernetes config file present at `$HOME/.kube/config`.

```
make run
```

### Build the Operator Image

Use the following commands to build the image of Neo4J operator and push to desired Docker repository.

```
IMG=weldnorthanalytics/neo4j-operator:v1.0.4 \
make docker-build docker-push
```

### Generate Deployment Manifests
```
rm -rf deploy
mkdir -p deploy

make generate manifests

cd config/manager && kustomize edit set image controller=weldnorthanalytics/neo4j-operator:v1.0.4
cd ../..
kubectl kustomize config/default > ./deploy/operator.yaml
```

### Direct Access to Neo4J Cluster

For debugging and development you might want to access the Neo4J cluster directly. For example, if you created the cluster with name `example1` in the `default` namespace, you can forward the Neo4J port to any of the pods (e.g. `neo4j-core-example1-0`) as follows:

    $ kubectl port-forward -n default neo4j-core-example1-0 7473:7473
