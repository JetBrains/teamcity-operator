package util

object E2eSteps {

    private const val kindVersion = "v0.33.0"
    private const val kindSha256 = "aee6151561422756b764a4ae28e7f44cda5af5a9eead3cc9985112b1de8d8e0d"
    private const val kubectlVersion = "v1.37.0"
    private const val kubectlSha256 = "6129359f4e1f3848a5572ccb0b26cf28b8ca08cef38c95a765b2f64a2c961a2f"

    val kindNodes = listOf(
        "1.26" to "v1.26.15@sha256:c79602a44b4056d7e48dc20f7504350f1e87530fe953428b792def00bc1076dd",
        "1.30" to "v1.30.13@sha256:397209b3d947d154f6641f2d0ce8d473732bd91c87d9575ade99049aa33cd648",
        "1.33" to "v1.33.7@sha256:d26ef333bdb2cbe9862a0f7c3803ecc7b4303d8cea8e814b481b09949d353040",
        "1.37" to "v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5",
    )

    fun installToolsScriptContent(): String = """
        #!/usr/bin/env bash
        set -euo pipefail
        rm -rf .e2e-bin .e2e-logs
        mkdir -p .e2e-bin .e2e-logs
        curl -sSfL --retry 3 -o .e2e-bin/kind https://github.com/kubernetes-sigs/kind/releases/download/$kindVersion/kind-linux-amd64
        curl -sSfL --retry 3 -o .e2e-bin/kubectl https://dl.k8s.io/release/$kubectlVersion/bin/linux/amd64/kubectl
        printf '%s  %s\n%s  %s\n' $kindSha256 .e2e-bin/kind $kubectlSha256 .e2e-bin/kubectl | sha256sum -c -
        chmod +x .e2e-bin/kind .e2e-bin/kubectl
        .e2e-bin/kind version
        .e2e-bin/kubectl version --client
    """.trimIndent()

    fun runScriptContent(): String = """
        #!/usr/bin/env bash
        set -euo pipefail
        export PATH="${'$'}PWD/.e2e-bin:${'$'}PATH"
        RC=0
        MATRIX="kind:%k8s.node%" PARALLEL=1 LOAD_PULLED_IMAGES=false LOGDIR="${'$'}PWD/.e2e-logs" bash test/e2e/k8s_matrix.sh || RC=${'$'}?
        for f in .e2e-logs/*.log; do
          echo "##teamcity[blockOpened name='${'$'}(basename "${'$'}f")']"
          sed 's/\x1b\[[0-9;]*m//g' "${'$'}f"
          echo "##teamcity[blockClosed name='${'$'}(basename "${'$'}f")']"
        done
        rm -f .e2e-logs/*.kubeconfig
        exit ${'$'}RC
    """.trimIndent()
}
