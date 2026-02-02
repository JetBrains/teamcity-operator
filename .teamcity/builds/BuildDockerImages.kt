package builds

import _Self.vcsRoots.TeamCityOperatorVCSRoot
import consts.dockerHubRegistryConnectionId
import consts.dockerImageName
import jetbrains.buildServer.configs.kotlin.BuildType
import jetbrains.buildServer.configs.kotlin.FailureAction
import jetbrains.buildServer.configs.kotlin.buildFeatures.dockerRegistryConnections
import jetbrains.buildServer.configs.kotlin.buildFeatures.perfmon
import jetbrains.buildServer.configs.kotlin.buildSteps.script
import jetbrains.buildServer.configs.kotlin.matrix

object BuildDockerImages : BuildType({
    name = "Build Docker Images"
    id("TeamCityOperatorBuildDockerImages")

    params {
        param("predicted_version", PrepareRelease.depParamRefs["predicted_version"].toString())
        param("docker_image", dockerImageName)
    }

    vcs {
        root(TeamCityOperatorVCSRoot)
    }

    dependencies {
        snapshot(PrepareRelease) {
            onDependencyFailure = FailureAction.FAIL_TO_START
            onDependencyCancel = FailureAction.FAIL_TO_START
        }
    }

    steps {
        script {
            name = "Build and push Docker image"
            id = "Build_and_push_docker_image"
            scriptContent = """
                echo "Building for architecture: %arch%"
                echo "Image tag: %docker_image%:%predicted_version%-%arch%"
                
                # Build single-platform image natively (no cross-compile needed)
                docker build -t %docker_image%:%predicted_version%-%arch% .
                docker push %docker_image%:%predicted_version%-%arch%
                
                echo "##teamcity[setParameter name='docker_image_with_arch' value='%docker_image%:%predicted_version%-%arch%']"
            """.trimIndent()
        }
    }


    features {
        matrix {
            param("arch", listOf(
                value("amd64", label = "AMD64"),
                value("arm64", label = "ARM64")
            ))
        }
        perfmon {}
        dockerRegistryConnections {
            loginToRegistry = on {
                dockerRegistryId = dockerHubRegistryConnectionId
            }
        }
    }
})
