|Branch|Status|
|:---:|:---:|
|master|[![Build status](https://ci.appveyor.com/api/projects/status/iqxooj0a4aepv1n1/branch/master?svg=true)](https://ci.appveyor.com/project/jfrog-ecosystem/jfrog-cli-go/branch/master) [![JFrog Pipelines](https://badgen.net/github/status/jfrog/jfrog-cli/master?label=JFrog%20Pipelines)](https://ecosysjfrog.jfrog.io/ui/pipelines/myPipelines/test_cli?branch=master)
|dev|[![Build status](https://ci.appveyor.com/api/projects/status/iqxooj0a4aepv1n1/branch/dev?svg=true)](https://ci.appveyor.com/project/jfrog-ecosystem/jfrog-cli-go/branch/dev) [![JFrog Pipelines](https://badgen.net/github/status/jfrog/jfrog-cli/dev?label=JFrog%20Pipelines)](https://ecosysjfrog.jfrog.io/ui/pipelines/myPipelines/test_cli?branch=dev)|

# Table of Contents
- [Overview](#overview)
- [Download and Installation](#download-and-installation)
- [Building the Executable](#building-the-executable)
- [Tests](#tests)
- [Code Contributions](#code-contributions)
- [Using JFrog CLI](#using-jfrog-cli)
- [JFrog CLI Plugins](#jfrog-cli-plugins)
- [Release Notes](#release-notes)

# Overview
JFrog CLI is a compact and smart client that provides a simple interface that automates access to *Artifactory*, *Bintray* and *Mission Control* through their respective REST APIs.
By using the JFrog CLI, you can greatly simplify your automation scripts making them more readable and easier to maintain.
Several features of the JFrog CLI makes your scripts more efficient and reliable:

- Multi-threaded upload and download of artifacts make builds run faster
- Checksum optimization reduces redundant file transfers
- Wildcards and regular expressions give you an easy way to collect all the artifacts you wish to upload or download.
- "Dry run" gives you a preview of file transfer operations before you actually run them

# Download and Installation

You can either install JFrog CLI using one of the supported installers or download its executable directly. Visit the [Install JFrog CLI Page](https://jfrog.com/getcli/) for details.

# Building the Executable

JFrog CLI is written in the [Go programming language](https://golang.org/), so to build the CLI yourself, you first need to have Go installed and configured on your machine.

## Install Go

To download and install `Go`, please refer to the [Go documentation](https://golang.org/doc/install).
Please download `Go 1.14.x` or above.

## Download and Build the CLI

Navigate to a directory where you want to create the jfrog-cli project, **outside** the `$GOPATH` tree.

If the `GOPATH` variable is unset, it's default value is the go folder under the user home.

Verify that the `GO111MODULE` variable is either unset, or explicitly set to `auto`.

Clone the jfrog-cli project by executing the following command:
````
git clone https://github.com/jfrog/jfrog-cli
````
Build the project by navigating to the jfrog folder and executing the following commands.
On Unix based systems run:
````
cd jfrog-cli
build/build.sh
````
On Windows run:
````
cd jfrog-cli
build\build.bat
````
Once completed, you will find the JFrog CLI executable at your current directory.

# Tests
### Usage
````
go test -v github.com/jfrog/jfrog-cli [test-types] [flags]
````

The flags are:

| Flag | Description |
| --- | --- |
| `-rt.url` | [Default: http://localhost:8081/artifactory] Artifactory URL. |
| `-rt.user` | [Default: admin] Artifactory username.|
| `-rt.password` | [Default: password] Artifactory password. |
| `-rt.apikey` | Artifactory API key. |
| `-rt.accessToken` | Artifactory access token. |

The types are:

| Type | Description |
| --- | --- |
| `-test.artifactory` | Artifactory tests |
| `-test.npm` | Npm tests |
| `-test.maven` | Maven tests |
| `-test.gradle` | Gradle tests |
| `-test.docker` | Docker tests |
| `-test.go` | Go tests |
| `-test.pip` | Pip tests |
| `-test.nuget` | Nuget tests |
| `-test.plugins` | Plugins tests |

* Running the tests will create builds and repositories with timestamps,
for example: `cli-tests-rt1-1592990748` and `cli-tests-rt2-1592990748`.<br/>
Once the tests are completed, the content of these repositories will be deleted.

#### Artifactory tests
In addition to [general optional flags](#Usage) you can use the following optional artifactory flags.

| Flag | Description |
| --- | --- |
| `-rt.sshKeyPath` | [Optional] Ssh key file path. Should be used only if the Artifactory URL format is ssh://[domain]:port. |
| `-rt.sshPassphrase` | [Optional] Ssh key passphrase. |


##### Examples
To run artifactory tests execute the following command.
````
go test -v github.com/jfrog/jfrog-cli -test.artifactory [flags]
````

#### Npm tests
##### Requirement
* The *npm* executables should be included as part of the *PATH* environment variable.
* The tests are compatible with npm 7 and higher.

##### Limitation
* Currently, npm integration support only http(s) connections to Artifactory using username and password.

##### Examples
To run npm tests execute the following command.
````
go test -v github.com/jfrog/jfrog-cli -test.npm [flags]
````

#### Maven tests
##### Requirements
* The *M2_HOME* environment variable should be set to the local maven installation path.
* The *java* executable should be included as part of the *PATH* environment variable. Alternatively, set the *JAVA_HOME* environment variable.

##### Limitation
* Currently, maven integration support only http(s) connections to Artifactory using username and password.

##### Examples
To run maven tests execute the following command.
````
go test -v github.com/jfrog/jfrog-cli -test.maven [flags]
````

#### Gradle tests
##### Requirements
* The *gradle* executables should be included as part of the *PATH* environment variable.
* The *java* executable should be included as part of the *PATH* environment variable. Alternatively, set the *JAVA_HOME* environment variable.

##### Limitation
* Currently, gradle integration support only http(s) connections to Artifactory using username and password.

##### Examples
To run gradle tests execute the following command.
````
go test -v github.com/jfrog/jfrog-cli -test.gradle [flags]
````

#### Docker tests
In addition to [general optional flags](#Usage) you *must* use the following docker flags.
##### Requirements
* On Linux machines, [Podman](https://podman.io/) tests will be running, so make sure it's available in the local path.

| Flag | Description |
| --- | --- |
| `-rt.dockerRepoDomain` | Artifactory Docker registry domain. |
| `-rt.dockerVirtualRepo` | Artifactory Docker virtual repository name. |
| `-rt.dockerRemoteRepo` | Artifactory Docker remote repository name. |
| `-rt.DockerLocalRepo` | Artifactory Docker local repository name. |

##### Examples
To run docker tests execute the following command (fill out the missing parameters as described below).
````
go test -v github.com/jfrog/jfrog-cli -test.docker -rt.dockerRepoDomain=DOCKER_DOMAIN -rt.DockerLocalRepo=DOCKER_LOCAL_REPO [flags]
````

#### Go commands tests
##### Examples
To run go tests run the following command.
````
go test -v github.com/jfrog/jfrog-cli -test.go [flags]
````

#### NuGet tests
##### Requirement
* Add NuGet executable to the system search path (PATH environment variable).
* Create a remote repository named jfrog-cli-tests-nuget-remote-repo.
* Run the following command.

##### Examples
````
go test -v github.com/jfrog/jfrog-cli -test.nuget [flags]
````

#### Pip tests
##### Requirement
* Add Python and pip executables to the system search path (PATH environment variable).
* Pip tests must run inside a clean pip-environment. You can either activate a virtual-environment and execute the tests from within, or provide the path to your virtual-environment using the -rt.pipVirtualEnv flag.
* Run the following command:

In addition to [general optional flags](#Usage) you can use the following optional pip flags

| Flag | Description |
| --- | --- |
| `-rt.pipVirtualEnv` | [Optional] Path to the directory of a clean pip virtual-environment. Make sure to provide the binaries directory (in unix: */bin*, in windows: *\Scripts*). |

##### Examples
````
go test -v github.com/jfrog/jfrog-cli -test.pip [flags]
````

#### Plugins tests
* To run Plugins tests execute the following command:
````
go test -v github.com/jfrog/jfrog-cli -test.plugins
````

### Bintray tests
Bintray tests credentials are taken from the CLI configuration. If non configured or not passed as flags, the tests will fail.

To run Bintray tests execute the following command:
````
go test -v github.com/jfrog/jfrog-cli -test.bintray
````
Flags:

| Flag | Description |
| --- | --- |
| `-bt.user` | [Mandatory if not configured] Bintray username. |
| `-bt.key` | [Mandatory if not configured] Bintray API key. |
| `-bt.org` | [Optional] Bintray organization. If not configured, *-bt.user* is used as the organization name. |

* Running the tests will create a repository named `cli-tests-bintray-<timestamp>` in bintray.<br/>
  Once the tests are completed, the repository will be deleted.

### Distribution tests
In addition to [general optional flags](#Usage) you can use the following flags:

| Flag | Description |
| --- | --- |
| `-rt.distUrl` | [Mandatory] JFrog Distribution URL. |
| `-rt.distAccessToken` | [Optional] Distribution access token. |

To run distribution tests run the following command:
```
go test -v github.com/jfrog/jfrog-cli -test.distribution [flags]
```

# Code Contributions
We welcome code contributions through pull requests from the community.
## Pull Requests Guidelines
* If the existing tests do not already cover your changes, please add tests..
* Pull requests should be created on the *dev* branch.
* Please use [gofmt](https://golang.org/cmd/gofmt/) for formatting the code before submitting the pull request.

# Using JFrog CLI
JFrog CLI can be used for a variety of functions with Artifactory, Bintray, Xray and Mission Control,
and has a dedicated set of commands for each product.
To learn how to use JFrog CLI, please visit the [JFrog CLI User Guide](https://www.jfrog.com/confluence/display/CLI/Welcome+to+JFrog+CLI).

# JFrog CLI Plugins
JFrog CLI plugins support enhancing the functionality of JFrog CLI to meet the specific user and organization needs. The source code of a plugin is maintained as an open source Go project on GitHub. All public plugins are registered in JFrog CLI's Plugins Registry, which is hosted in the [jfrog-cli-plugins-reg](https://github.com/jfrog/jfrog-cli-plugins-reg) GitHub repository. We encourage you, as developers, to create plugins and share them publically with the rest of the community. Read more about this in the [JFrog CLI Plugin Developer Guide](guides/jfrog-cli-plugins-developer-guide.md).

# Release Notes
The release notes are available [here](RELEASE.md#release-notes).
