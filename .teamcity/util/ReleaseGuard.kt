package util

import consts.dockerImageName

object ReleaseGuard {

    private const val publicRepoUrl = "https://github.com/JetBrains/teamcity-operator.git"

    private fun checkFunctions(): String = """
        remote_tag_sha() {
          local out rc=0
          out=${'$'}(git ls-remote --exit-code --tags "$publicRepoUrl" "refs/tags/${'$'}1") || rc=${'$'}?
          case "${'$'}rc" in
            0) echo "${'$'}out" | awk '{print ${'$'}1}' ;;
            2) echo "" ;;
            *) echo "Cannot check git tag ${'$'}1 (git ls-remote exit ${'$'}rc)" >&2; exit 1 ;;
          esac
        }
        refuse_existing_image() {
          local out
          if out=${'$'}(docker buildx imagetools inspect "${'$'}1" 2>&1); then
            echo "Image ${'$'}1 is already published; refusing to overwrite it." >&2
            exit 1
          fi
          if ! echo "${'$'}out" | grep -qiE 'not found|manifest unknown'; then
            echo "Cannot check image ${'$'}1: ${'$'}out" >&2
            exit 1
          fi
          echo "Image ${'$'}1 is not published yet"
        }
    """.trimIndent()

    fun refuseExistingVersionScriptContent(): String = """
        #!/usr/bin/env bash
        set -euo pipefail
        ${checkFunctions().prependIndent("        ").trimStart()}
        VERSION="%predicted_version%"
        TAG_SHA=${'$'}(remote_tag_sha "${'$'}VERSION")
        if [ -n "${'$'}TAG_SHA" ]; then
          echo "Release ${'$'}VERSION already has a git tag; another release chain got there first." >&2
          exit 1
        fi
        refuse_existing_image "$dockerImageName:${'$'}VERSION"
        refuse_existing_image "$dockerImageName:${'$'}VERSION-amd64"
        refuse_existing_image "$dockerImageName:${'$'}VERSION-arm64"
    """.trimIndent()

    fun reserveTagScriptContent(): String = """
        #!/usr/bin/env bash
        set -euo pipefail
        ${checkFunctions().prependIndent("        ").trimStart()}
        VERSION="%predicted_version%"
        HEAD_SHA=${'$'}(git rev-parse HEAD)
        TAG_SHA=${'$'}(remote_tag_sha "${'$'}VERSION")
        if [ "${'$'}TAG_SHA" = "${'$'}HEAD_SHA" ]; then
          echo "Tag ${'$'}VERSION already points at ${'$'}HEAD_SHA; continuing a rerun of this release."
          exit 0
        fi
        if [ -n "${'$'}TAG_SHA" ]; then
          echo "Tag ${'$'}VERSION points at ${'$'}TAG_SHA, not ${'$'}HEAD_SHA; another release chain got there first." >&2
          exit 1
        fi
        refuse_existing_image "$dockerImageName:${'$'}VERSION"
        mkdir -p ~/.ssh && chmod 700 ~/.ssh
        ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts
        git remote set-url origin git@github.com:JetBrains/teamcity-operator.git
        git push origin "${'$'}HEAD_SHA:refs/tags/${'$'}VERSION"
    """.trimIndent()
}
