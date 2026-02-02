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
import util.BuildSteps

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
            scriptContent = BuildSteps.predictTagScriptContent()
        }
        script {
            name = "Run tests"
            id = "Run_tests"
            scriptContent = BuildSteps.runTestsScriptContent()
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
