import _Self.vcsRoots.TeamCityOperatorVCSRoot
import builds.BuildRelease
import consts.dockerHubRegistryConnectionId
import environment.EnvironmentProvider
import jetbrains.buildServer.configs.kotlin.*
import jetbrains.buildServer.configs.kotlin.projectFeatures.dockerRegistry

version = "2025.11"


project {
    vcsRoot(TeamCityOperatorVCSRoot)
    buildType(BuildRelease)

    features {
        dockerRegistry {
            id = dockerHubRegistryConnectionId
            name = "Docker Registry"
            userName = EnvironmentProvider.dockerRegistryUsername()
            password = EnvironmentProvider.dockerRegistryPassword()
        }
    }
}
