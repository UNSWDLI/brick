<!-- omit in toc -->
# brick: Building

- [Project README](../README.md)

<!-- omit in toc -->
## Table of contents

- [Requirements](#requirements)
  - [Building source code](#building-source-code)
  - [Running](#running)
- [Instructions](#instructions)

## Requirements

### Building source code

- Go 1.13+
- GCC
  - if building with custom options (as the provided `Makefile` does)
- `make`
  - if using the provided `Makefile`

### Running

The `brick` application has been tested with:

- Go 1.13+
- Ubuntu Linux 16.04, 18.04

## Instructions

1. [Download](https://golang.org/dl/) Go
1. [Install](https://golang.org/doc/install) Go
   - NOTE: Pay special attention to the remarks about `$HOME/.profile`
1. Clone the repo
   1. `cd /tmp`
   1. `git clone https://github.com/atc0005/brick`
   1. `cd brick`
1. Install dependencies (optional)
   - for Ubuntu Linux
     - `sudo apt-get install make gcc`
   - for CentOS Linux
     - `sudo yum install make gcc`
   - for Windows
     - Emulated environments (*easier*)
       - Skip all of this and build using the default `go build` command in
         Windows (see below for use of the `-mod=vendor` flag)
       - build using Windows Subsystem for Linux Ubuntu environment and just
         copy out the Windows binaries from that environment
       - If already running a Docker environment, use a container with the Go
         tool-chain already installed
       - If already familiar with LXD, create a container and follow the
         installation steps given previously to install required dependencies
     - Native tooling (*harder*)
       - see the StackOverflow Question `32127524` link in the
         [References](references.md) section for potential options for
         installing `make` on Windows
       - see the mingw-w64 project homepage link in the
         [References](references.md) section for options for installing `gcc`
         and related packages on Windows
1. Build binaries
   - for the current operating system
     - `go build -mod=vendor ./cmd/brick/`
     - `go build -mod=vendor ./cmd/es/`
     - `go build -mod=vendor ./cmd/ezproxy/`
       - *forces build to use bundled dependencies in top-level `vendor`
         folder*
   - for all supported platforms (where `make` is installed)
      - `make all`
   - for use on Windows
      - `make windows`
   - for use on Linux
     - `make linux`
1. Copy the newly compiled binaries from the applicable paths below and deploy
   using the instructions provided in our [deployment doc](deploy.md).
   - if using `Makefile`:
     - `/tmp/brick/release_assets/brick/`
     - `/tmp/brick/release_assets/es/`
     - `/tmp/brick/release_assets/ezproxy/`
   - if using separate `go build` invocations
     - `/tmp/brick/`
