package _Self

import _Self.vcsRoots.TeamCityOperatorVCSRoot
import jetbrains.buildServer.configs.kotlin.Project
import projects.TeamCityOperator.TeamCityOperator

object Shared : Project({
    name = "Shared"

    vcsRoot(TeamCityOperatorVCSRoot)
    subProject(TeamCityOperator)
})
