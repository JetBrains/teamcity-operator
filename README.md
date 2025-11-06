# TeamCity Kubernetes Operator

TeamCity Operator is a Kubernetes operator that deploys and manages TeamCity servers using a Custom Resource Definition (CRD).

- Helm chart: charts/teamcity-operator
- CRD group/version: jetbrains.com/v1beta1, kind: TeamCity
- Sample manifests: config/samples/v1beta1
- Development guide: docs/DEVELOPMENT.md

## Installation

### Prerequisites

- Kubernetes v1.26.0+
- cert-manager installed in the cluster
  - Tested with cert-manager v1.14.3
  - Install example:
    ```shell
    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.3/cert-manager.yaml
    ```

### Install the Operator via Helm

Install release 0.0.20 of the chart from GitHub Releases:

```shell
helm upgrade --install teamcity-operator \
  -n teamcity-operator --create-namespace \
  https://github.com/JetBrains/teamcity-operator/releases/download/0.0.20/teamcity-operator-0.0.20.tgz
```

#### Install from source

From the repository root of this project:

```shell
helm upgrade --install teamcity-operator \
  -n teamcity-operator --create-namespace \
  ./charts/teamcity-operator
```

Verify installation:
```shell
kubectl get pods -n teamcity-operator
kubectl get crd teamcities.jetbrains.com
```

## Create a TeamCity server

After installing the operator, apply a TeamCity custom resource. Below are common configurations. You can find additional samples under config/samples/v1beta1.

Note: When setting CPU and memory requests/limits for nodes, consult the official TeamCity Server Requirements to estimate appropriate resources: https://www.jetbrains.com/help/teamcity/system-requirements.html#TeamCity+Server+Requirements.

### Standalone TeamCity Main Node with a data directory

```yaml
apiVersion: jetbrains.com/v1beta1
kind: TeamCity
metadata:
  name: teamcity
  namespace: default
  finalizers:
    - "teamcity.jetbrains.com/finalizer"
spec:
  image: jetbrains/teamcity-server
  mainNode:
    name: main-node
    spec:
      requests:
        cpu: "400m"
        memory: "2500Mi"
  dataDirVolumeClaim:
    name: teamcity-data-dir
    volumeMount:
      name: teamcity-data-dir
      mountPath: /storage
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
```

### Standalone TeamCity Main Node with a pre-configured external database

```yaml
apiVersion: v1
data:
  connectionProperties.password: DB_PASSWORD
  connectionProperties.user: DB_USER
  connectionUrl: DB_CONNECTION_STRING # format jdbc:mysql://DB_HOST:DB_PORT/SCHEMA_NAME
kind: Secret
metadata:
  name: database-properties
---
apiVersion: jetbrains.com/v1beta1
kind: TeamCity
metadata:
  name: teamcity-with-database
  namespace: default
  finalizers:
    - "teamcity.jetbrains.com/finalizer"
spec:
  image: jetbrains/teamcity-server
  databaseSecret:
    secret: database-properties
  mainNode:
    name: main-node
    spec:
      env:
        AWS_DEFAULT_REGION: "eu-west-1"
      requests:
        cpu: "400m"
        memory: "1000Mi"
  dataDirVolumeClaim:
    name: teamcity-data-dir
    volumeMount:
      name: teamcity-data-dir
      mountPath: /storage
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
```

### Standalone TeamCity Main Node with startup properties

```yaml
apiVersion: jetbrains.com/v1beta1
kind: TeamCity
metadata:
  name: teamcity-with-startup-properties
  namespace: default
  finalizers:
    - "teamcity.jetbrains.com/finalizer"
spec:
  image: jetbrains/teamcity-server
  startupPropertiesConfig:
    teamcity.startup.maintenance: "false"
    teamcity.firstStart.setupAdmin.enabled: "false"
  mainNode:
    name: main-node
    spec:
      requests:
        cpu: "400m"
        memory: "1000Mi"
  dataDirVolumeClaim:
    name: teamcity-data-dir
    volumeMount:
      name: teamcity-data-dir
      mountPath: /storage
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
```

### Standalone TeamCity Main Node with Ingress and Service

```yaml
apiVersion: jetbrains.com/v1beta1
kind: TeamCity
metadata:
  name: teamcity-sample-with-ingress
  namespace: default
  finalizers:
    - "teamcity.jetbrains.com/finalizer"
spec:
  image: jetbrains/teamcity-server
  mainNode:
    name: main-node
    spec:
      requests:
        cpu: "400m"
        memory: "1000Mi"
  dataDirVolumeClaim:
    name: teamcity-data-dir
    volumeMount:
      name: teamcity-data-dir
      mountPath: /storage
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
  serviceList:
    - name: main-node
      spec:
        selector:
          app.kubernetes.io/name: teamcity-sample-with-ingress
          app.kubernetes.io/component: teamcity-server
          app.kubernetes.io/part-of: teamcity
        ports:
          - protocol: TCP
            port: 8111
            targetPort: 8111
        clusterIP: None
  ingressList:
  - name: teamcity-sample-with-ingress
    annotations:
      nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
      nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
      nginx.ingress.kubernetes.io/server-snippets: |
        location / {
          proxy_http_version 1.1;
          proxy_set_header X-Forwarded-Host $http_host;
          proxy_set_header X-Forwarded-Proto $scheme;
          proxy_set_header X-Forwarded-For $remote_addr;
          proxy_set_header Host $host;
          proxy_set_header Upgrade $http_upgrade; # WebSocket support
          proxy_set_header Connection "upgrade"; # WebSocket support
        }
    spec:
      ingressClassName: nginx
      rules:
        - host: teamcity.mycompany.com
          http:
            paths:
              - backend:
                  service:
                    name: main-node
                    port:
                      number: 8111
                pathType: ImplementationSpecific
```

### Standalone TeamCity Main Node with a ServiceAccount

```yaml
apiVersion: jetbrains.com/v1beta1
kind: TeamCity
metadata:
  name: teamcity-with-service-account
  namespace: default
  finalizers:
    - "teamcity.jetbrains.com/finalizer"
spec:
  image: jetbrains/teamcity-server
  serviceAccount:
    name: teamcity-service-account
    annotations:
      eks.amazonaws.com/role-arn: AWS_ROLE
  mainNode:
    name: node
    spec:
      requests:
        cpu: "400m"
        memory: "1000Mi"
  dataDirVolumeClaim:
    name: teamcity-data-dir
    volumeMount:
      name: teamcity-data-dir
      mountPath: /storage
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
```

### Standalone TeamCity Main Node with init containers

```yaml
apiVersion: jetbrains.com/v1beta1
kind: TeamCity
metadata:
  name: teamcity-init-containers
  namespace: default
  finalizers:
    - "teamcity.jetbrains.com/finalizer"
spec:
  image: jetbrains/teamcity-server
  mainNode:
    name: main-node
    spec:
      initContainers:
        - name: init-myservice
          image: busybox:1.28
          command: [ 'sh', '-c', "echo Hello" ]
      requests:
        cpu: "400m"
        memory: "1000Mi"
  dataDirVolumeClaim:
    name: teamcity-data-dir
    volumeMount:
      name: teamcity-data-dir
      mountPath: /storage
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
```

### Multi-node: Main Node with one Secondary TeamCity Node without responsibilities

```yaml
apiVersion: jetbrains.com/v1beta1
kind: TeamCity
metadata:
  name: teamcity-sample-with-secondary-node-read-only
  namespace: default
  finalizers:
    - "teamcity.jetbrains.com/finalizer"
spec:
  image: jetbrains/teamcity-server
  dataDirVolumeClaim:
    name: teamcity-data-dir
    volumeMount:
      name: teamcity-data-dir
      mountPath: /storage
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
  mainNode:
    name: main-node
    spec:
      requests:
        cpu: "1000m"
        memory: "2500Mi"
  secondaryNodes:
    - name: secondary-node
      spec:
        requests:
          cpu: "1000m"
          memory: "2500Mi"
```

### Multi-node: Main Node with two Secondary TeamCity Nodes with responsibilities

See TeamCity multi-node responsibilities: https://www.jetbrains.com/help/teamcity/multinode-setup.html#Responsibilities

```yaml
apiVersion: jetbrains.com/v1beta1
kind: TeamCity
metadata:
  name: teamcity-sample-with-secondary-node
  namespace: default
  finalizers:
    - "teamcity.jetbrains.com/finalizer"
spec:
  image: jetbrains/teamcity-server
  startupPropertiesConfig:
    teamcity.startup.maintenance: "false"
    teamcity.firstStart.setupAdmin.enabled: "false"
  dataDirVolumeClaim:
    name: teamcity-data-dir
    volumeMount:
      name: teamcity-data-dir
      mountPath: /storage
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
  mainNode:
    name: main-node
    annotations:
      cluster-autoscaler.kubernetes.io/safe-to-evict: "true"
    spec:
      requests:
        cpu: "1000m"
        memory: "2500Mi"
      responsibilities: [ "MAIN_NODE", "CAN_PROCESS_BUILD_MESSAGES", "CAN_CHECK_FOR_CHANGES", "CAN_PROCESS_BUILD_TRIGGERS", "CAN_PROCESS_USER_DATA_MODIFICATION_REQUESTS"]
  secondaryNodes:
    - name: secondary-node-0
      annotations:
        cluster-autoscaler.kubernetes.io/safe-to-evict: "false"
      spec:
        requests:
          cpu: "1000m"
          memory: "2500Mi"
        responsibilities: ["CAN_PROCESS_BUILD_MESSAGES", "CAN_CHECK_FOR_CHANGES", "CAN_PROCESS_BUILD_TRIGGERS", "CAN_PROCESS_USER_DATA_MODIFICATION_REQUESTS"]
    - name: secondary-node-1
      annotations:
        cluster-autoscaler.kubernetes.io/safe-to-evict: "false"
      spec:
        requests:
          cpu: "1000m"
          memory: "2500Mi"
        responsibilities: [ "CAN_PROCESS_BUILD_MESSAGES", "CAN_CHECK_FOR_CHANGES", "CAN_PROCESS_BUILD_TRIGGERS", "CAN_PROCESS_USER_DATA_MODIFICATION_REQUESTS" ]
```

### Zero-downtime upgrades

Enables a safe upgrade flow where the operator restarts or replaces nodes in a way that keeps TeamCity available. In short: it upgrades nodes one at a time and ensures at least one node is serving the UI during the process.

### How it works

- Standalone Main Node setup
    - The operator temporarily creates a Secondary TeamCity Node using the Main Node’s spec.
    - Traffic keeps going to this temporary node while the Main Node is restarted/upgraded.
    - After the Main Node is healthy again, the temporary node is removed.

- Multi-node setup (Main Node + Secondary TeamCity Nodes)
    - Nodes are upgraded sequentially (for example, one Secondary TeamCity Node at a time, then the Main Node), so at least one node continues to serve requests.
    - The operator waits for a node to become healthy before moving on to the next one.

### What to keep in mind
- You annotate your `TeamCity` resource with `teamcity.jetbrains.com/update-policy: zero-downtime` to turn this on.
- This flow assumes your deployment can support multiple nodes briefly running side-by-side (e.g., using a shared database) so the UI remains available during upgrades.


```yaml
apiVersion: jetbrains.com/v1beta1
kind: TeamCity
metadata:
  name: teamcity-with-zero-downtime-upgrade
  namespace: default
  finalizers:
    - "teamcity.jetbrains.com/finalizer"
  annotations:
    teamcity.jetbrains.com/update-policy: zero-downtime
spec:
  image: jetbrains/teamcity-server
  databaseSecret:
    secret: database-properties
  startupPropertiesConfig:
    teamcity.startup.maintenance: "false"
    teamcity.firstStart.setupAdmin.enabled: "false"
  dataDirVolumeClaim:
    name: teamcity-data-dir
    volumeMount:
      name: teamcity-data-dir
      mountPath: /storage
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
  mainNode:
    name: main-node
    spec:
      requests:
        cpu: "1000m"
        memory: "2500Mi"
```

## Migration

- Migrating from an existing TeamCity installation? See [docs/MIGRATION.md](docs/MIGRATION.md) for two approaches:
  - Approach 1: Move the TeamCity Data Directory to the Operator-managed PVC (simplest).
  - Approach 2: Full backup and restore into a new, empty database.

## Contributing

- Development and local debugging instructions have been moved to [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).
- Issues and PRs are welcome.


