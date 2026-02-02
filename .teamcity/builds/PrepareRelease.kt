package builds

import _Self.vcsRoots.TeamCityOperatorVCSRoot
import jetbrains.buildServer.configs.kotlin.BuildType
import jetbrains.buildServer.configs.kotlin.buildFeatures.commitStatusPublisher
import jetbrains.buildServer.configs.kotlin.buildFeatures.perfmon
import jetbrains.buildServer.configs.kotlin.buildSteps.script
import util.BuildSteps

object PrepareRelease : BuildType({
    name = "Prepare Release"
    id("TeamCityOperatorPrepareRelease")

    params {
        param("predicted_version", "0.0.0")
        param("docker_image", "jetbrains/teamcity-operator")
    }

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
