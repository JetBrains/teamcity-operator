package builds

import _Self.vcsRoots.TeamCityOperatorVCSRoot
import consts.dockerImageName
import jetbrains.buildServer.configs.kotlin.BuildType
import jetbrains.buildServer.configs.kotlin.CheckoutMode
import jetbrains.buildServer.configs.kotlin.buildFeatures.commitStatusPublisher
import jetbrains.buildServer.configs.kotlin.buildFeatures.perfmon
import jetbrains.buildServer.configs.kotlin.buildSteps.script
import util.BuildSteps
import util.OfflineTestSteps

object PrepareRelease : BuildType({
    name = "Prepare Release"
    id("TeamCityOperatorPrepareRelease")

    vcs {
        root(TeamCityOperatorVCSRoot)
        checkoutMode = CheckoutMode.ON_AGENT
    }

    steps {
        script {
            name = "Predict tag"
            id = "Predict_tag"
            scriptContent = BuildSteps.predictTagScriptContent()
        }
        script {
            name = "Prefetch dependencies (Hermeto)"
            id = "Prefetch_dependencies"
            scriptContent = OfflineTestSteps.prefetchScriptContent()
        }
        script {
            name = "Run tests offline"
            id = "Run_tests"
            scriptContent = OfflineTestSteps.offlineTestScriptContent()
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
