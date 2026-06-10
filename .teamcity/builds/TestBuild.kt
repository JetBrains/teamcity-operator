package builds

import _Self.vcsRoots.TeamCityOperatorVCSRoot
import jetbrains.buildServer.configs.kotlin.BuildType
import jetbrains.buildServer.configs.kotlin.buildFeatures.PullRequests
import jetbrains.buildServer.configs.kotlin.buildFeatures.commitStatusPublisher
import jetbrains.buildServer.configs.kotlin.buildFeatures.perfmon
import jetbrains.buildServer.configs.kotlin.buildFeatures.pullRequests
import jetbrains.buildServer.configs.kotlin.buildSteps.script
import jetbrains.buildServer.configs.kotlin.triggers.vcs
import util.BuildSteps

object TestBuild : BuildType({
    name = "Test&Build"
    id("TeamCityOperatorTestBuild")

    vcs {
        root(TeamCityOperatorVCSRoot)
        branchFilter = """
            +:pull/*
            +:refs/heads/main
        """.trimIndent()
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
