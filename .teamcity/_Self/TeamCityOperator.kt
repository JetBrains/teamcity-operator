package _Self

import _Self.vcsRoots.TeamCityOperatorVCSRoot
import jetbrains.buildServer.configs.kotlin.Project
import projects.TeamCityOperator.TeamCityOperator

object TeamCityOperator : Project({
    name = "TeamCity Operator"

    vcsRoot(TeamCityOperatorVCSRoot)
    subProject(TeamCityOperator)
})
