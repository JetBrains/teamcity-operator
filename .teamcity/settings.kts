import jetbrains.buildServer.configs.kotlin.*

/*
 * TeamCity versioned settings entry point.
 * Project hierarchy is defined in _Self/Shared.kt.
 * Credentials are read from DSL context parameters (see .teamcity/README-DSL-CONTEXT.md).
 */

version = "2025.11"

project(_Self.TeamCityOperator)
