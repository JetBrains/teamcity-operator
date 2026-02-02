package builds

import _Self.vcsRoots.TeamCityOperatorVCSRoot
import consts.dockerHubRegistryConnectionId
import consts.dockerImageName
import environment.EnvironmentProvider
import jetbrains.buildServer.configs.kotlin.BuildType
import jetbrains.buildServer.configs.kotlin.FailureAction
import jetbrains.buildServer.configs.kotlin.buildFeatures.commitStatusPublisher
import jetbrains.buildServer.configs.kotlin.buildFeatures.dockerRegistryConnections
import jetbrains.buildServer.configs.kotlin.buildFeatures.perfmon
import jetbrains.buildServer.configs.kotlin.buildFeatures.sshAgent
import jetbrains.buildServer.configs.kotlin.buildSteps.script
import jetbrains.buildServer.configs.kotlin.triggers.vcs

object FinalizeRelease : BuildType({
    name = "Finalize Release"
    id("TeamCityOperatorFinalizeRelease")

    params {
        param("predicted_version", PrepareRelease.depParamRefs["predicted_version"].toString())
        param("docker_image", dockerImageName)
    }

    vcs {
        root(TeamCityOperatorVCSRoot)
    }

    dependencies {
        snapshot(BuildDockerImages) {
            onDependencyFailure = FailureAction.FAIL_TO_START
            onDependencyCancel = FailureAction.FAIL_TO_START
        }
    }

    triggers {
        vcs {
            branchFilter = "+:<default>"
        }
    }

    steps {
        script {
            name = "Create and push multi-arch manifest"
            id = "Create_multiarch_manifest"
            scriptContent = """
                echo "Creating multi-arch manifest for %docker_image%:%predicted_version%"
                
                docker buildx imagetools create -t %docker_image%:%predicted_version% \
                    %docker_image%:%predicted_version%-amd64 \
                    %docker_image%:%predicted_version%-arm64
                
                echo "##teamcity[setParameter name='docker_image_full' value='%docker_image%:%predicted_version%']"
                echo "Multi-arch manifest pushed successfully"
            """.trimIndent()
        }
        script {
            name = "Push git tags"
            id = "Push_tags"
            scriptContent = """
                git remote set-url origin git@github.com:JetBrains/teamcity-operator.git
                git tag %predicted_version%
                git push --tags
            """.trimIndent()
        }
        script {
            name = "Label build"
            id = "Label_build"
            scriptContent = """
                curl -s --user "%system.teamcity.auth.userId%:%system.teamcity.auth.password%" \
                    --request POST \
                    --header "Content-Type: application/xml" \
                    --data "<tags><tag name='%predicted_version%'/></tags>" \
                    "%teamcity.serverUrl%/httpAuth/app/rest/builds/id:%teamcity.build.id%/tags/"
            """.trimIndent()
        }
    }

    features {
        perfmon {}
        sshAgent {
            teamcitySshKey = EnvironmentProvider.teamcityOperatorSshKeyName()
        }
        dockerRegistryConnections {
            loginToRegistry = on {
                dockerRegistryId = dockerHubRegistryConnectionId
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
