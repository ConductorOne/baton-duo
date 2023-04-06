# baton-duo
`baton-duo` is a connector for Duo built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It communicates with the Duo API to sync data about users and groups and admins.

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

## Prerequisites

1. Duo Beyond, Duo Access, or Duo MFA plan with `Owner` role. 
2. Protect an Application called `Admin API`. It can be found in `Applications` in` Duo Admin Panel`. Save your integration key, secret key, and API hostname.
3. Grant permissions: 
  - Grant administrators
  - Grant read information
  - Grant applications
  - Grant read resource

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-duo
baton-duo
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_SECRET_KEY=secretKey BATON_INTEGRATION_KEY=integrationKey BATON_API_HOSTNAME=apiHostname ghcr.io/conductorone/baton-duo:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-duo/cmd/baton-duo@main

BATON_SECRET_KEY=secretKey BATON_INTEGRATION_KEY=integrationKey BATON_API_HOSTNAME=apiHostname
baton resources
```

# Data Model

`baton-duo` pulls down information about the following Duo resources:
- Users
- Groups
- Admins

# Contributing, Support, and Issues

We started Baton because we were tired of taking screenshots and manually building spreadsheets. We welcome contributions, and ideas, no matter how small -- our goal is to make identity and permissions sprawl less painful for everyone. If you have questions, problems, or ideas: Please open a Github Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-duo` Command Line Usage

```
baton-duo

Usage:
  baton-duo [flags]
  baton-duo [command]

Available Commands:
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --api-hostname string      Duo api hostname key needed to complete the setup to connect to the Duo API. ($BATON_API_HOSTNAME)
  -f, --file string              The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                     help for baton-duo
      --integration-key string   Duo integration key needed to complete the setup to connect to the Duo API. ($BATON_INTEGRATION_KEY)
      --log-format string        The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string         The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
      --secret-key string        Duo secret key needed to complete the setup to connect to the Duo API. ($BATON_SECRET_KEY)
  -v, --version                  version for baton-duo

Use "baton-duo [command] --help" for more information about a command.

```