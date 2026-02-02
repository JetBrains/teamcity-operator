package projects.TeamCityOperator

import environment.EnvironmentProvider
import jetbrains.buildServer.configs.kotlin.Project
import jetbrains.buildServer.configs.kotlin.projectFeatures.dockerRegistry
import projects.TeamCityOperator.builds.BuildRelease

object TeamCityOperator : Project({
    name = "TeamCity Operator"

    buildType(BuildRelease)

    features {
        dockerRegistry {
            id = "PROJECT_EXT_24"
            name = "Docker Registry"
            userName = EnvironmentProvider.dockerRegistryUsername()
            password = EnvironmentProvider.dockerRegistryPassword()
        }
    }
})
