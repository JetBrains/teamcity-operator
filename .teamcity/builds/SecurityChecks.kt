package builds

import _Self.vcsRoots.TeamCityOperatorVCSRoot
import jetbrains.buildServer.configs.kotlin.BuildType
import jetbrains.buildServer.configs.kotlin.CheckoutMode
import jetbrains.buildServer.configs.kotlin.buildFeatures.PullRequests
import jetbrains.buildServer.configs.kotlin.buildFeatures.commitStatusPublisher
import jetbrains.buildServer.configs.kotlin.buildFeatures.perfmon
import jetbrains.buildServer.configs.kotlin.buildFeatures.pullRequests
import jetbrains.buildServer.configs.kotlin.buildSteps.script
import util.OfflineTestSteps

object SecurityChecks : BuildType({
    name = "Security Checks"
    id("TeamCityOperatorSecurityChecks")

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
            name = "Prefetch dependencies (Hermeto)"
            id = "Prefetch_dependencies"
            scriptContent = OfflineTestSteps.prefetchScriptContent()
        }
        script {
            name = "Fetch security tools and vulnerability database"
            id = "Fetch_security_tools"
            scriptContent = OfflineTestSteps.fetchSecurityToolsScriptContent()
        }
        script {
            name = "Security checks offline (govulncheck, licenses)"
            id = "Security_checks"
            scriptContent = OfflineTestSteps.securityChecksScriptContent()
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
