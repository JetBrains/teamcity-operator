import _Self.vcsRoots.TeamCityOperatorVCSRoot
import builds.BuildDockerImages
import builds.E2eMatrix
import builds.FinalizeRelease
import builds.PrepareRelease
import builds.SecurityChecks
import builds.TestBuild
import consts.dockerHubRegistryConnectionId
import environment.EnvironmentProvider
import jetbrains.buildServer.configs.kotlin.*
import jetbrains.buildServer.configs.kotlin.projectFeatures.dockerRegistry

version = "2025.11"

project {
    vcsRoot(TeamCityOperatorVCSRoot)

    buildType(SecurityChecks)
    buildType(PrepareRelease)
    buildType(BuildDockerImages)
    buildType(FinalizeRelease)
    buildType(TestBuild)
    buildType(E2eMatrix)

    features {
        dockerRegistry {
            id = dockerHubRegistryConnectionId
            name = "Docker Registry"
            userName = EnvironmentProvider.dockerRegistryUsername()
            password = EnvironmentProvider.dockerRegistryPassword()
        }
    }
}
