# Upgrade Policy

This is the contract every operator release keeps for existing installations. Pull requests and release checks are reviewed against it.

## Supported upgrade paths

- Upgrading from either of the two previous releases to the current release is supported. For example, `0.0.26` → `0.0.28` and `0.0.27` → `0.0.28`.
- Older releases upgrade one supported step at a time.
- Downgrades are not supported.

## Supported Kubernetes versions

- Kubernetes v1.26.0 and later, with cert-manager installed (see [README](../README.md#prerequisites)).
- Raising the minimum Kubernetes version is a breaking change and must be in the release notes.

## CRD rules (`jetbrains.com/v1beta1`, kind `TeamCity`)

1. No field is removed or renamed.
2. New fields are optional, and leaving them unset keeps the previous behaviour.
3. A default never changes meaning. If an existing default must change, add a new field instead.
4. Validation is never tightened for objects that already exist. A new rule applies only to new objects or to fields that change (ratcheting), so existing `TeamCity` resources keep passing admission.
5. Field types and allowed enum values are never narrowed.

## Operator upgrade and running TeamCity pods

- Upgrading the operator must not restart TeamCity pods.
- Any change to the generated pod template (StatefulSet spec, environment, volumes, probes, labels or annotations on the pod) restarts pods. A release with such a change says so in the release notes, together with which nodes restart and why.
- Changes to other generated resources (Services, Ingress, PVCs, ConfigMaps) are listed in the release notes when they change existing objects.
- The operator never deletes or recreates a PersistentVolumeClaim with data during an upgrade.

## Deprecation

1. A field or behaviour that will be removed first gets a deprecation warning from the admission webhook for at least one release.
2. The release notes list the deprecation and the replacement.
3. Removal happens in a later release and counts as a breaking change.

## Breaking changes

A release that cannot keep these rules is a breaking release. It must have:

- A "Breaking changes" section in the release notes.
- Manual upgrade steps, tested from the two previous releases.
