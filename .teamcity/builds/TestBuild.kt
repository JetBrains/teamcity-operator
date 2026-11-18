package builds

import _Self.vcsRoots.TeamCityOperatorVCSRoot
import jetbrains.buildServer.configs.kotlin.BuildType
import jetbrains.buildServer.configs.kotlin.CheckoutMode
import jetbrains.buildServer.configs.kotlin.buildFeatures.PullRequests
import jetbrains.buildServer.configs.kotlin.buildFeatures.commitStatusPublisher
import jetbrains.buildServer.configs.kotlin.buildFeatures.perfmon
import jetbrains.buildServer.configs.kotlin.buildFeatures.pullRequests
import jetbrains.buildServer.configs.kotlin.buildSteps.script
import jetbrains.buildServer.configs.kotlin.triggers.vcs
import util.BuildSteps
import util.OfflineTestSteps

object TestBuild : BuildType({
    name = "Test&Build"
    id("TeamCityOperatorTestBuild")

    params {
        param("expected.sha", "")
    }

    vcs {
        root(TeamCityOperatorVCSRoot)
        checkoutMode = CheckoutMode.ON_AGENT
        branchFilter = """
            +:pull/*
            +:refs/heads/main
        """.trimIndent()
    }

    steps {
        script {
            name = "Verify reviewed commit"
            id = "Verify_reviewed_commit"
            scriptContent = OfflineTestSteps.verifyCommitScriptContent()
        }
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

    triggers {
        // No automatic builds. Tests are queued only via GitHub Actions on /test comment.
        vcs {
            branchFilter = """
                -:*
            """.trimIndent()
        }
    }

    features {
        perfmon {}
        pullRequests {
            vcsRootExtId = TeamCityOperatorVCSRoot.id?.toString()
            provider = github {
                authType = vcsRoot()
                filterTargetBranch = "+:refs/heads/main"
                filterAuthorRole = PullRequests.GitHubRoleFilter.EVERYBODY
            }
        }
        commitStatusPublisher {
            vcsRootExtId = TeamCityOperatorVCSRoot.id?.toString()
            publisher = github {
                githubUrl = "https://api.github.com"
                authType = vcsRoot()
            }
        }
    }
})
