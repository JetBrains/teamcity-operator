# TeamCity Kubernetes Operator


TeamCity Operator is a Kubernetes operator that deploys and manages TeamCity servers using a Custom Resource Definition (CRD).

- Helm chart: charts/teamcity-operator
- CRD group/version: jetbrains.com/v1beta1, kind: TeamCity
- Sample manifests: `config/samples/v1beta1` (see also `config/samples/teamcity_full.yaml`)
- Annotations reference: see [Annotations](#annotations) below
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

Install release 0.0.21 of the chart from GitHub Releases:

```shell
helm upgrade --install teamcity-operator \
  -n teamcity-operator --create-namespace \
  https://github.com/JetBrains/teamcity-operator/releases/download/0.0.21/teamcity-operator-0.0.21.tgz
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

After installing the operator, apply a TeamCity custom resource. Below are common configurations.

Each manifest under `config/samples/v1beta1/` includes a header comment with an `kubectl apply` command. For a combined reference manifest, see `config/samples/teamcity_full.yaml`.

| Sample | Purpose |
|--------|---------|
| `_v1beta1_teamcity.yaml` | Minimal single-node |
| `_v1beta1_teamcity_with_database.yaml` | External database (bundled demo MySQL) |
| `_v1beta1_teamcity_with_startup_properties.yaml` | Startup properties ConfigMap |
| `_v1beta1_teamcity_with_ingress.yaml` | Headless Service + Ingress |
| `_v1beta1_teamcity_with_serviceaccount.yaml` | Dedicated ServiceAccount |
| `_v1beta1_teamcity_with_init_containers.yaml` | Custom init containers |
| `_v1beta1_teamcity_with_node_selector.yaml` | Node selector |
| `_v1beta1_teamcity_with_affinity.yaml` | Pod affinity |
| `_v1beta1_teamcity_with_service.yaml` | Headless Service + `serviceName` (new deployment) |
| `_v1beta1_teamcity_with_service_name_recreate.yaml` | `serviceName` change on existing deployment |
| `_v1beta1_teamcity_with_secondary_node.yaml` | Multi-node with responsibilities |
| `_v1beta1_teamcity_with_secondary_node_read_only.yaml` | Secondary node without responsibilities |
| `_v1beta1_teamcity_with_zero_downtime_upgrade.yaml` | Zero-downtime upgrade (single node) |
| `_v1beta1_teamcity_with_secondary_node_with_zero_downtime_upgrade.yaml` | Zero-downtime upgrade (multi-node) |

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
        cpu: "900m"
        memory: "1512Mi"
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
        cpu: "900m"
        memory: "1512Mi"
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
        cpu: "900m"
        memory: "1512Mi"
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
        cpu: "900m"
        memory: "1512Mi"
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
    - name: tc-sample-for-ingress
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
  - name: tc-ingress
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
                    name: tc-sample-for-ingress
                    port:
                      number: 8111
                pathType: ImplementationSpecific
```

The Ingress backend `service.name` must match a name from `spec.serviceList`. Service selectors must use `app.kubernetes.io/component: teamcity-server` — that is the label the operator sets on TeamCity pods.

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
        cpu: "900m"
        memory: "1512Mi"
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
        cpu: "900m"
        memory: "1512Mi"
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

Multi-node setups require a shared database. Provide `spec.databaseSecret` pointing at a Secret with JDBC settings (see the database example above). The read-only sample references an external Secret; see `config/samples/v1beta1/_v1beta1_teamcity_with_secondary_node_read_only.yaml`.

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
  databaseSecret:
    secret: database-properties
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
        cpu: "900m"
        memory: "1512Mi"
  secondaryNodes:
    - name: secondary-node
      spec:
        requests:
          cpu: "900m"
          memory: "1512Mi"
```

When no node sets `responsibilities`, the operator accepts the configuration as-is. Once any node specifies responsibilities, validation rules apply (see below).

### Multi-node: Main Node with one Secondary TeamCity Node with responsibilities

See TeamCity multi-node responsibilities: https://www.jetbrains.com/help/teamcity/multinode-setup.html#Responsibilities

When responsibilities are configured, the main node must include at least `MAIN_NODE` and `CAN_PROCESS_USER_DATA_MODIFICATION_REQUESTS`. Secondary nodes must not include `MAIN_NODE`. All five responsibilities should be distributed across the cluster without duplication:

- `MAIN_NODE`
- `CAN_PROCESS_BUILD_MESSAGES`
- `CAN_CHECK_FOR_CHANGES`
- `CAN_PROCESS_BUILD_TRIGGERS`
- `CAN_PROCESS_USER_DATA_MODIFICATION_REQUESTS`

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
        cpu: "900m"
        memory: "1512Mi"
      responsibilities: [ "MAIN_NODE", "CAN_PROCESS_USER_DATA_MODIFICATION_REQUESTS" ]
  secondaryNodes:
    - name: secondary-node
      annotations:
        cluster-autoscaler.kubernetes.io/safe-to-evict: "false"
      spec:
        requests:
          cpu: "900m"
          memory: "1512Mi"
        responsibilities: [ "CAN_PROCESS_BUILD_MESSAGES", "CAN_CHECK_FOR_CHANGES", "CAN_PROCESS_BUILD_TRIGGERS" ]
```

### Headless Service and per-node serviceName

Set `spec.serviceList` with `clusterIP: None` and reference the service from each node via `spec.mainNode.spec.serviceName` (and/or `spec.secondaryNodes[].spec.serviceName`). This gives each StatefulSet pod a stable DNS name under the headless service.

For a **new** TeamCity created with `serviceName` already in the spec, no extra annotation is required. See `config/samples/v1beta1/_v1beta1_teamcity_with_service.yaml`.

```yaml
  serviceList:
    - name: tc-sample-one-svc
      spec:
        selector:
          app.kubernetes.io/name: teamcity-sample-with-svc
          app.kubernetes.io/component: teamcity-server
          app.kubernetes.io/part-of: teamcity
        ports:
          - protocol: TCP
            port: 8111
            targetPort: 8111
        clusterIP: None
  mainNode:
    name: node
    spec:
      serviceName: tc-sample-one-svc
      requests:
        cpu: "400m"
        memory: "1000Mi"
  secondaryNodes:
    - name: secondary-node
      spec:
        serviceName: tc-sample-one-svc
        requests:
          cpu: "400m"
          memory: "1000Mi"
```

### Changing serviceName on an existing deployment

Kubernetes treats `StatefulSet.spec.serviceName` as immutable. If you add or change `serviceName` on a TeamCity whose StatefulSets were already created without it, set the [`allow-sts-recreate`](#teamcity-resource-metadata) annotation before applying the change.

Workflow:

1. `kubectl annotate teamcity <name> teamcity.jetbrains.com/allow-sts-recreate=true`
2. Patch or edit the TeamCity and set `spec.mainNode.spec.serviceName` (and/or secondary node `serviceName`)
3. The operator recreates the affected StatefulSet(s) and restarts the node(s)

Without the annotation, the webhook rejects the change and the controller reports the conflict in status and events. See `config/samples/v1beta1/_v1beta1_teamcity_with_service_name_recreate.yaml`.

### Zero-downtime upgrades

Note: This is an experimental feature. Behavior and configuration may change between releases. We welcome feedback — please open an issue in this repository with your experience and suggestions.

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
- Enable zero-downtime upgrades with the [`update-policy`](#teamcity-resource-metadata) annotation (see [Annotations](#annotations)).
- This flow assumes your deployment can support multiple nodes briefly running side-by-side (e.g., using a shared database) so the UI remains available during upgrades.
- Sample manifests bundle a demo MySQL Deployment and `database-properties` Secret. Apply only one such bundle per namespace, or use your own database and Secret.
- Full samples: `config/samples/v1beta1/_v1beta1_teamcity_with_zero_downtime_upgrade.yaml` (single node) and `config/samples/v1beta1/_v1beta1_teamcity_with_secondary_node_with_zero_downtime_upgrade.yaml` (multi-node).


```yaml
apiVersion: jetbrains.com/v1beta1
kind: TeamCity
metadata:
  name: teamcity-sample-zero-downtime
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

## Annotations

The operator uses annotations on the TeamCity custom resource to control upgrade and recreate behavior. Annotations on fields under `spec` are copied to the corresponding Kubernetes objects (StatefulSet pod templates, Services, Ingresses, PVCs, and so on).

### TeamCity resource metadata

These annotations are set on the TeamCity CR itself (`metadata.annotations`):

| Key | Value | When to use | Effect |
|-----|-------|-------------|--------|
| `teamcity.jetbrains.com/update-policy` | `zero-downtime` | Optional. Upgrading image or spec while keeping the UI available. | Operator performs a rolling, one-node-at-a-time upgrade. On a single-node setup it temporarily adds a secondary node; on multi-node setups it upgrades secondaries first, then the main node. Requires a shared database. **Experimental** — see [Zero-downtime upgrades](#zero-downtime-upgrades). |
| `teamcity.jetbrains.com/allow-sts-recreate` | `"true"` | Required when adding or changing `spec.*.serviceName` on an existing TeamCity. | Webhook allows the change; operator deletes and recreates affected StatefulSet(s) and restarts the node(s). Without this annotation the update is rejected. See [Changing serviceName on an existing deployment](#changing-servicename-on-an-existing-deployment). |

Example:

```yaml
metadata:
  annotations:
    teamcity.jetbrains.com/update-policy: zero-downtime
    teamcity.jetbrains.com/allow-sts-recreate: "true"
```

Only set `allow-sts-recreate` when you intend to recreate StatefulSets; it is not needed for new TeamCity resources created with `serviceName` already in the spec.

### Finalizers

| Key | When to use | Effect |
|-----|-------------|--------|
| `teamcity.jetbrains.com/finalizer` | Recommended on every TeamCity CR (`metadata.finalizers`). | On delete, the operator runs cleanup (for example, zero-downtime checkpoint removal) before the CR is removed. Include this finalizer in your manifests — all samples do. |

### Annotations on spec fields

The operator passes these through to child resources without modification:

| Spec field | Applied to |
|------------|------------|
| `spec.mainNode.annotations` | Pod template of the main node StatefulSet |
| `spec.secondaryNodes[].annotations` | Pod template of each secondary node StatefulSet |
| `spec.serviceList[].annotations` | Matching Service |
| `spec.ingressList[].annotations` | Matching Ingress |
| `spec.serviceAccount.annotations` | TeamCity ServiceAccount |
| `spec.dataDirVolumeClaim.annotations` | Data directory PVC |
| `spec.persistentVolumeClaims[].annotations` | Additional PVCs |

Use these for integration with other cluster components — for example, `cluster-autoscaler.kubernetes.io/safe-to-evict` on node annotations, NGINX Ingress proxy settings on `ingressList` entries, or `eks.amazonaws.com/role-arn` on `serviceAccount` (see the ServiceAccount example above).

### Labels applied by the operator

These are **labels** (not annotations), set automatically on StatefulSets and pods; you normally do not set them on the TeamCity CR:

| Label | Meaning |
|-------|---------|
| `teamcity.jetbrains.com/node-name` | Node name from the spec (`mainNode.name` or secondary node name) |
| `teamcity.jetbrains.com/role` | `main` or `secondary` |

When defining `spec.serviceList` selectors, use the standard labels the operator sets: `app.kubernetes.io/name` (TeamCity CR name), `app.kubernetes.io/component: teamcity-server`, and `app.kubernetes.io/part-of: teamcity`.

## Migration

- Migrating from an existing TeamCity installation? See [docs/MIGRATION.md](docs/MIGRATION.md) for two approaches:
  - Approach 1: Move the TeamCity Data Directory to the Operator-managed PVC (simplest).
  - Approach 2: Full backup and restore into a new, empty database.

## Limitations

- Custom volume mounts are not available; this feature is under development.
- Mounting custom Secrets as environment variables is currently not supported.
- Ingress integration has been tested with the NGINX Ingress Controller only.

## Contributing

- Development and local debugging instructions have been moved to [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).
- Issues and PRs are welcome.




