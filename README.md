<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [nfy](#nfy)
  - [Install](#install)
  - [Introduction](#introduction)
    - [The Problem](#the-problem)
    - [The Goal](#the-goal)
  - [Basic Example](#basic-example)
    - [Build Container Image](#build-container-image)
      - [Advanced Example](#advanced-example)
  - [Parallelism](#parallelism)
  - [Recipes](#recipes)
  - [Code Structure](#code-structure)
    - [Import Statements](#import-statements)
  - [Dependencies](#dependencies)
    - [Target Evaluation](#target-evaluation)
    - [Target Overloading](#target-overloading)
      - [Use Cases](#use-cases)
    - [Locking](#locking)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# nfy

`nfy` is an **experimental** configuration management that aims to u**n**i**fy** the concept of the `Dockerfile`,
the install script, and the configuration manager.

You can think of it as a bridge between bare metal configuration and containers.

There are no compatibility or reliability guarantees until we're v1.0.0.

## Install
```
go get -u cdr.dev/nfy/cmd/nfy
```

## Introduction

### The Problem
1. **Dockerfiles don't scale.**
    1. As Dockerfiles become large, developers begin moving installation to scripts and installers, undermining layer caching.
        1. Smaller files are more readable. 
        1. Small scripts can be tested independently.
1. **Dockerfiles are constrained by their linear import graph.**
    1. Software tool images (e.g rust, nginx) force an operating system on the user.
    1. Multiple images can't be used at the same time.
        1. If you want an image with `go` and `rust`, you use one as your `FROM`, and write
        `RUN` statements to simulate what the image already does.
1. **Dockerfiles can't be applied to the local system.**
    1. Development workstations are long-lived and bare metal, where production servers are ephemeral and containerized.
    Software management diverges greatly between the two.
    
### The Goal
1. **Containerization**
    - Build container images from your install scripts.
    - Update software in environments without recreating the environment.
        - Run `nfy install` to apply the configuration to your local machine.
        - Run `nfy build` to create a container with the install scripts.
1. **Speed**
    - Automatically parallelize installation.
    - Stop duplication computation by coupling each install target with an explicit check.
1. **Portability**
    - Install the same target on different systems based on what requirements are available.
1. **Reusability**
    - Share installers and configuration via GitHub imports.
    - Use some of your personal dotfiles on your development machine or workstation.

## Basic Example

In order to install `wget` and vim, create a file called `nfy.yml` with the following contents:

```yaml
wget:
  comment: "wget lets us grab files from HTTP servers."
  install: "apt-get -y install wget"
  check: "wget -h"
# Delegate vim to Ammar's personal dotfiles.
vim:
    deps:
      - github.com/ammario/dotfiles:vim
```

then run `sudo nfy install`, which installs every target by default.

If `wget` is already installed, the `check` step will pass and `install` won't run.

### Build Container Image

Run `sudo nfy build -b ubuntu nfy-ubuntu` to build an Ubuntu container image called `nfy-ubuntu` with `wget` and my vim
configuration.

The Dockerfile looks like:

```
FROM ubuntu
# wget: wget lets us grab files from HTTP servers.
RUN apt-get -y install wget
```

#### Advanced Example

```yaml
apt-get:
  check: "apt-get -h"
apt-update:
  comment: "Ensure the package cache is up to date."
  install: "apt-get update -y"
  build_only: true
  deps:
    - apt-get
apt:
  deps:
    - apt-get
    - apt-update
htop:
  install: "apt-get install -y htop"
  check: "htop -h"
  deps:
    - apt
wget:
  install: "apt-get install -y wget"
  check: "wget -h"
  deps:
    - apt
```

produces

```Dockerfile
FROM ubuntu
# Ensure the "apt-get" dependency exists:
RUN apt-get -h
# apt-update: Ensure the package cache is up to date.
RUN apt-get update -y
RUN apt-get install -y wget
RUN apt-get install -y htop
```
(those comments are generated automatically)

## Parallelism

`nfy` creates a tree of files and external dependencies rooted in your `nfy.yaml`. The tree is a directed acyclic graph
and evaluated with automatic parallelism. Take this example (`X -> Y` means X depends on Y):

```
A -> B
A -> C
B -> D
```

Is converted into 2 threads that look like:

```
Compute D
Wait C
Compute B
```

```
Compute C
Wait B
Compute A
```

This parallelism is active during apply and build, meaning `nfy` can perform much faster than Docker.

## Recipes
Each recipe has a name or a _target_. For example:

```
curl:
    install: "..."
```

has the target `curl`.

The target can be used to reference the recipe later.

Recipes support the following parameters:

| Name | Usage |
| ---- | ----- |
| install |  An executable command or script to install. |
| check |  An executable command or script location to check if installation is necessary. |
| fast_install |  If "yes", indicates that the install is fast enough and running check is unnecessary. |
| deps |  A list of targets which must exist before this can install. |
| build_only | Specify whether command will only run in container builds. |
| comment | Include a comment in the Dockerfile. |
| files |  A list of files which must be available in the working directory. |

A target must implement one of `check` or `install`. A target with `install` and no `check` will print a warning when
it is evaluated, unless `fast_install` is set.

A target with a `check` but no install can be used to represent hard requirements, such as

```yaml
# Ensures apt-get exists
apt-get:
    check: "apt-get -h"
```

## Code Structure

### Import Statements
An `import` directive loads in recipes from a file. The configuration is evaluated in order, so the list of imports
must be before the imported target is referenced. Example:

```yaml
import:
    - "apt.yml"
htop:
    install: "sudo apt -y install htop"
    deps:
      - apt
```

Every import is processed before recipe evaluation begins.

You can also import entire directories via globbing. For example:

```yaml
import:
    - "nfy/*.yml"
```

## Dependencies
Dependencies can exist on a per recipe and per file basis. Dependencies on a file are automatically added to each
recipe in the file.

Dependencies in the `nfy.yml` file can make hard, global requirements about the system, for example:

```yaml
apt-get:
    check: "apt-get -h"
deps:
    - apt-get
```

### Target Evaluation

A target can be provided in one of three formats:

- `wget` references a local target somewhere in the source tree
- `github.com/user/repo:wget` references a remote target named `wget` hosted on git.

### Target Overloading
What if you want to install `htop` in your Macbook or Linux server?

```yaml
htop:
    check: "htop -h"
    install: "apt-get install -h htop"
deps:
    - apt-get
```

will obviously fail because apt-get is not installed. Instead, you should _overload_ the target with multiple
installers. For example:

```yaml
htop:
  check: "htop -h"
  install_apt:
    script: "apt-get install -y htop"
    deps:
      - apt
  install_brew:
    script: "brew install htop"
    deps:
      - brew

```

We cannot simply provide multiple `install` directives because it is illegal YAML for keys to conflict.

The suffix is nice for debugging nfy execution, too.

#### Use Cases

- Targetting multiple operating systems
- Tolerating different dependencies (e.g using "curl" instead of "wget")

#### Docker
The first target is always used in Docker images. The first is assumed the default.

### Locking

**Unimplemented**

Each external dependency is locked to a particular commit in the `nfy.lock` file.

You can update the repository with `nfy update github.com/<user>/<repo>@<tag>`.

The locking mechanism offers security and stability to your config. You should check in your lock file.
