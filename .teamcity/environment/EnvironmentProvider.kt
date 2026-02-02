package environment

import jetbrains.buildServer.configs.kotlin.DslContext

object EnvironmentProvider {
    fun dockerRegistryUsername(): String = DslContext.getParameter("docker.registry.username")
    fun dockerRegistryPassword(): String = DslContext.getParameter("docker.registry.password")

    fun githubUsername(): String = DslContext.getParameter("github.username")
    fun githubPassword(): String = DslContext.getParameter("github.password")

    fun teamcityOperatorSshKeyName(): String = DslContext.getParameter("teamcity.operator.ssh.key.name", "teamcity-operator-key")
}
