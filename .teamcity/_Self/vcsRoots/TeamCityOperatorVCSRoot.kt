package _Self.vcsRoots

import environment.EnvironmentProvider
import jetbrains.buildServer.configs.kotlin.vcs.GitVcsRoot

object TeamCityOperatorVCSRoot : GitVcsRoot({
    name = "https://github.com/JetBrains/teamcity-operator.git"
    url = "https://github.com/JetBrains/teamcity-operator"
    branch = "refs/heads/main"
    branchSpec = "refs/heads/*"
    authMethod = password {
        userName = EnvironmentProvider.githubUsername()
        password = EnvironmentProvider.githubPassword()
    }
    param("pipelines.connectionId", "tc-cloud-github-connection")
})
