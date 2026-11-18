package util

object OfflineTestSteps {

    const val hermetoImage =
        "ghcr.io/hermetoproject/hermeto@sha256:b941a5258887a47deb3d741fa4a3183ec7ec3a99c1c4d0be9aa10c1d15fe0b9a"
    const val goImage =
        "golang@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea"

    private const val envtestVersion = "1.26.0"
    private const val envtestSha512Amd64 =
        "115275a461e5fb09515f4d87a12b07ccdf1662ed79a89af085ec965af1a2e9dd5ddcfa9399da0c409de29b7f2aede29d9af6cb91eeb80bebbeb09dc56a9ab36f"
    private const val envtestSha512Arm64 =
        "7703a8a5072bf8744436036e744cb93f79d91d49d3de5b407cdd59ee49894a3461a1eeb1bf4fe7afc4b2a2d505c62c058d5410fda95cf93a1a6fab8fd33a8687"

    fun verifyCommitScriptContent(): String = """
        set -eu
        EXPECTED="%expected.sha%"
        ACTUAL="%build.vcs.number%"
        BRANCH="%teamcity.build.branch%"
        case "${'$'}BRANCH" in
          pull/*)
            if [ -z "${'$'}EXPECTED" ]; then
              echo "##teamcity[buildProblem description='Pull request builds must be queued with expected.sha set to the reviewed commit']"
              exit 1
            fi
            ;;
        esac
        if [ -n "${'$'}EXPECTED" ] && [ "${'$'}EXPECTED" != "${'$'}ACTUAL" ]; then
          echo "##teamcity[buildProblem description='Checked out ${'$'}ACTUAL but the reviewed commit is ${'$'}EXPECTED']"
          exit 1
        fi
        echo "Building reviewed commit ${'$'}ACTUAL"
    """.trimIndent()

    fun prefetchScriptContent(): String = """
        #!/usr/bin/env bash
        set -euo pipefail
        case "${'$'}(uname -m)" in
          x86_64) ARCH=amd64; SHA512=$envtestSha512Amd64 ;;
          aarch64|arm64) ARCH=arm64; SHA512=$envtestSha512Arm64 ;;
          *) echo "Unsupported agent architecture ${'$'}(uname -m)"; exit 1 ;;
        esac
        rm -rf .hermeto-input .hermeto-output .hermeto.env
        mkdir -p .hermeto-input
        cat > .hermeto-input/artifacts.lock.yaml <<EOF
        metadata:
          version: "1.0"
        artifacts:
          - download_url: https://github.com/kubernetes-sigs/controller-tools/releases/download/envtest-v$envtestVersion/envtest-v$envtestVersion-linux-${'$'}ARCH.tar.gz
            checksum: sha512:${'$'}SHA512
            filename: envtest.tar.gz
        EOF
        git remote set-url origin https://github.com/JetBrains/teamcity-operator.git
        HERMETO="docker run --rm -v ${'$'}PWD:/src -w /src $hermetoImage"
        ${'$'}HERMETO fetch-deps --source /src --output /src/.hermeto-output \
          '{"packages":[{"type":"gomod","path":"."},{"type":"generic","path":".","lockfile":"/src/.hermeto-input/artifacts.lock.yaml"}]}'
        ${'$'}HERMETO generate-env /src/.hermeto-output -o /src/.hermeto.env --for-output-dir /src/.hermeto-output
        ${'$'}HERMETO inject-files /src/.hermeto-output --for-output-dir /src/.hermeto-output
        echo "##teamcity[publishArtifacts '.hermeto-output/bom.json => sbom']"
    """.trimIndent()

    fun offlineTestScriptContent(): String = """
        #!/usr/bin/env bash
        set -euo pipefail
        cat > .offline-test.sh <<'EOF'
        set -euo pipefail
        if curl -sS --max-time 5 -o /dev/null https://proxy.golang.org 2>/dev/null; then
          echo "##teamcity[buildProblem description='Network is reachable from the offline test container']"
          exit 1
        fi
        echo "Network is unreachable, as expected"
        . ./.hermeto.env
        export GOTOOLCHAIN=local GOFLAGS=-mod=mod
        mkdir -p /tmp/envtest
        tar -xzf .hermeto-output/deps/generic/envtest.tar.gz -C /tmp/envtest
        KUBEBUILDER_ASSETS="${'$'}(dirname "${'$'}(find /tmp/envtest -type f -name kube-apiserver | head -1)")"
        export KUBEBUILDER_ASSETS
        go version
        go vet ./...
        go run github.com/onsi/ginkgo/v2/ginkgo --teamcity-report report.out --coverprofile=cover.out -coverpkg=./... -r ./...
        EOF
        RC=0
        docker run --rm --network=none -v "${'$'}PWD":/src -w /src $goImage bash /src/.offline-test.sh || RC=${'$'}?
        cat report.out 2>/dev/null || true
        exit ${'$'}RC
    """.trimIndent()
}
