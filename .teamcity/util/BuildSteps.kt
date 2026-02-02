package util

object BuildSteps {

    fun predictTagScriptContent(): String = """
        go install github.com/restechnica/semverbot/cmd/sbot@v1.1.0
        sbot update version
        PREDICTED_NEXT_VERSION=${'$'}(sbot predict version)
        BRANCH=%teamcity.build.branch%
        BUILD_NUMBER=%build.number%
        if [ "%teamcity.build.branch%" = "main" ]; then
          VERSION_WITH_SUFFIX="${'$'}PREDICTED_NEXT_VERSION"
        else
          VERSION_WITH_SUFFIX="${'$'}PREDICTED_NEXT_VERSION-rc.%build.number%"
        fi
        echo "Artifact tag is ${'$'}VERSION_WITH_SUFFIX"
        echo "##teamcity[setParameter name='predicted_version' value='${'$'}VERSION_WITH_SUFFIX']"
    """.trimIndent()

    fun runTestsScriptContent(): String = """
        go version
        go install github.com/onsi/ginkgo/v2/ginkgo
        make test
        cat report.out
    """.trimIndent()
}
