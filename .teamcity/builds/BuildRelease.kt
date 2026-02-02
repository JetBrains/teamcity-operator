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

object BuildRelease : BuildType({
    name = "Build&Release"
    id("TeamCityOperatorBuildRelease")

    params {
        param("predicted_version", "0.0.0")
        param("platform", "linux/amd64,linux/arm64")
        param("docker_image", "jetbrains/teamcity-operator")
    }

    vcs {
        root(TeamCityOperatorVCSRoot)
        branchFilter = "+:<default>"
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
        script {
            name = "Build docker image"
            id = "Build_docker_image"
            scriptContent = """
                docker buildx create --name multiplatform || true
                docker buildx use multiplatform
                docker buildx build -t %docker_image%:%predicted_version% --push --platform %platform% .
                echo "##teamcity[setParameter name='docker_image_full' value='%docker_image%:%predicted_version%']"
            """.trimIndent()
        }
        script {
            name = "Push tags"
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
            scriptContent = """curl -s --user "%system.teamcity.auth.userId%:%system.teamcity.auth.password%" --request POST --header "Content-Type: application/xml" --data "<tags><tag name='%predicted_version%'/></tags>" "%teamcity.serverUrl%/httpAuth/app/rest/builds/id:%teamcity.build.id%/tags/""""
        }
    }

    triggers {
        vcs {
            branchFilter = "+:<default>"
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
