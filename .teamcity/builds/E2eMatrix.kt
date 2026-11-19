package builds

import _Self.vcsRoots.TeamCityOperatorVCSRoot
import jetbrains.buildServer.configs.kotlin.BuildType
import jetbrains.buildServer.configs.kotlin.CheckoutMode
import jetbrains.buildServer.configs.kotlin.FailureAction
import jetbrains.buildServer.configs.kotlin.ReuseBuilds
import jetbrains.buildServer.configs.kotlin.buildFeatures.PullRequests
import jetbrains.buildServer.configs.kotlin.buildFeatures.commitStatusPublisher
import jetbrains.buildServer.configs.kotlin.buildFeatures.perfmon
import jetbrains.buildServer.configs.kotlin.buildFeatures.pullRequests
import jetbrains.buildServer.configs.kotlin.buildSteps.script
import jetbrains.buildServer.configs.kotlin.matrix
import util.E2eSteps
import util.OfflineTestSteps

object E2eMatrix : BuildType({
    name = "E2E Kubernetes Matrix"
    id("TeamCityOperatorE2eMatrix")

    params {
        param("expected.sha", "")
    }

    vcs {
        root(TeamCityOperatorVCSRoot)
        checkoutMode = CheckoutMode.ON_AGENT
        branchFilter = """
            +:pull/*
            +:<default>
        """.trimIndent()
    }

    dependencies {
        snapshot(TestBuild) {
            onDependencyFailure = FailureAction.FAIL_TO_START
            onDependencyCancel = FailureAction.FAIL_TO_START
            reuseBuilds = ReuseBuilds.NO
        }
    }

    steps {
        script {
            name = "Verify reviewed commit"
            id = "Verify_reviewed_commit"
            scriptContent = OfflineTestSteps.verifyCommitScriptContent()
        }
        script {
            name = "Install pinned kind and kubectl"
            id = "Install_e2e_tools"
            scriptContent = E2eSteps.installToolsScriptContent()
        }
        script {
            name = "Upgrade check and integration tests on kind"
            id = "Run_e2e"
            scriptContent = E2eSteps.runScriptContent()
        }
    }

    artifactRules = ".e2e-logs => e2e-logs"

    failureConditions {
        executionTimeoutMin = 90
    }

    requirements {
        equals("teamcity.agent.jvm.os.name", "Linux")
        equals("teamcity.agent.cpuArchitecture", "x86_64")
        noLessThan("teamcity.agent.hardware.cpuCount", "8")
        noLessThan("teamcity.agent.hardware.memorySizeMb", "15000")
    }

    features {
        matrix {
            param("k8s.node", E2eSteps.kindNodes.map { (label, image) -> value(image, label = label) })
        }
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
