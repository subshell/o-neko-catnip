# O-Neko URL Trigger

[![CircleCI](https://circleci.com/gh/subshell/o-neko-url-trigger/tree/master.svg?style=svg)](https://circleci.com/gh/subshell/o-neko-url-trigger/tree/master)
[![Docker Image Version (latest semver)](https://img.shields.io/docker/v/subshellgmbh/o-neko-url-trigger?color=2496ED&label=subshellgmbh%2Fo-neko-url-trigger&logo=docker&logoColor=white&sort=semver)](https://hub.docker.com/r/subshellgmbh/o-neko-url-trigger/tags)

This is an optional extension application for [O-Neko](https://github.com/subshell/o-neko/).
Its purpose is to be used as a default backend / default ingress for stopped O-Neko deployments. It will then try to start the deployment 
with the URL it has been loaded with and will then redirect the user to the deployment once it started.

## Development Setup

Ideally you're able to deploy this application to Kubernetes and have a running O-Neko test instance at hand to connect this tool to.

### Required Tools

* Go >=1.17
* Docker
* Make
* UPX
