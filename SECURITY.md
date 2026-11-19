# Security Policy

## Supported versions

The operator is released from `main` as `0.0.x`. Security fixes go into the next release only.

| Version | Supported |
| --- | --- |
| Latest release | Yes |
| Older releases | No, upgrade to the latest release |

## Reporting a vulnerability

Do not open a public issue, pull request or discussion for a security problem.

Report it privately through GitHub: open the [Security tab](https://github.com/JetBrains/teamcity-operator/security) and click **Report a vulnerability**.

Include:

- The operator version or commit.
- The Kubernetes version and how the operator was installed (Helm chart or manifests).
- Steps to reproduce, and what an attacker can do.

## What to expect

- We acknowledge the report within 5 business days.
- We send an assessment and a planned fix date within 10 business days.
- We publish a GitHub security advisory when the fix is released, and credit you unless you ask us not to.

## Scope

In scope: the operator code, its container image, the Helm chart, the CRDs and the manifests in this repository.

Out of scope: TeamCity itself. Report TeamCity issues to JetBrains as described at https://www.jetbrains.com/privacy-security/.
