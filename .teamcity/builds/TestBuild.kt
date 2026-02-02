package builds

import _Self.vcsRoots.TeamCityOperatorVCSRoot
import consts.dockerHubRegistryConnectionId
import environment.EnvironmentProvider
import jetbrains.buildServer.configs.kotlin.BuildType
import jetbrains.buildServer.configs.kotlin.buildFeatures.commitStatusPublisher
import jetbrains.buildServer.configs.kotlin.buildFeatures.dockerRegistryConnections
import jetbrains.buildServer.configs.kotlin.buildFeatures.perfmon
import jetbrains.buildServer.configs.kotlin.buildFeatures.sshAgent
import jetbrains.buildServer.configs.kotlin.buildSteps.script
import jetbrains.buildServer.configs.kotlin.triggers.vcs

object TestBuild : BuildType({
    name = "Test&Build"
    id("TeamCityOperatorTestBuild")


    vcs {
        root(TeamCityOperatorVCSRoot)
    }

    steps {
        script {
            name = "Predict tag"
            id = "Predict_tag"
            scriptContent = """
                go install github.com/restechnica/semverbot/cmd/sbot@v1.1.0
                sbot update version
                PREDICTED_NEXT_VERSION=${'$'}(sbot predict version)
                echo "Artifact tag is ${'$'}PREDICTED_NEXT_VERSION" 
                echo "##teamcity[setParameter name='predicted_version' value='${'$'}PREDICTED_NEXT_VERSION']"
            """.trimIndent()
        }
        script {
            name = "Run tests"
            id = "Run_tests"
            scriptContent = """
                go version
                go install github.com/onsi/ginkgo/v2/ginkgo
                make test
                cat report.out
            """.trimIndent()
            dockerImage = "golang:1.22.0"
        }
    }

    triggers {
        vcs {
            branchFilter = """
                -:*
            """.trimIndent()
        }
    }

    features {
        perfmon {}
        commitStatusPublisher {
            vcsRootExtId = TeamCityOperatorVCSRoot.id?.toString()
            publisher = github {
                githubUrl = "https://api.github.com"
                authType = vcsRoot()
            }
        }
    }
})
