Ding is a simple secure self-hosted private build server (for continuous
integration) for individual developers and small teams, with specialized
features for building Go applications, intended to run on BSDs too.

Ding lets you configure repositories and build scripts and compile your
software.

For a build, ding:

- Fetches the sources code, with a git or hg clone, or with a script of your
  choosing.
- Sets up directories to run the build, under a unique UID, so builds don't
  interfere with each other or the system.
- Runs the build script, capturing output and parsing it for instructions
  referencing the version of the software built, releasable files and test
  coverage.
- Keeps track of the releasable files, allowing you to mark them as released so
  they are not cleaned up automatically.

Ding starts a web server with a UI for creating/configuring repositories,
starting builds, seeing the output. Ding serves an API at /ding/, including
real-time updates to builds and repositories. The web UI uses only this API.

Old builds are automatically cleaned up.

Command "ding kick" can be used in a git hook to signal that a build should
start. Gitea, github and bitbucket webhooks are also supported.

Command "ding build" can be used locally to run a build script similar to how
it would run on the build server. This allows testing build scripts without
pushing any commits. It clones the git or hg repository of the current working
directory, sets up environment variables and destination build directories, and
calls the provided build script. Optionally isolated using bubblewrap (bwrap)
on Linux.

Command "ding go" updates $HOME/sdk to the latest Go toolchains (current,
previous and optionaly next), and sets symlinks go/goprev/gonext.


# Installing

You should be up and running within a few minutes with the quickstart.

You'll need:

- a BSD or Linux machine
- git/mercurial or any other version control software you want to use

Ding is normally started as root. Ding then immediately starts an unprivileged
process under a ding-specific uid/gid for webserving. The privileged process
starts builds under a unique uids/gids, for isolation.

To install, first create a ding user:

	useradd -m ding

Download the "ding" binary to the ding home directory (see "Download" below).

Run the quickstart as root:

	./ding quickstart

The quickstart creates initial directories, fixes up permissions, and writes a
ding.conf configuration file.

On linux, you'll also get a systemd service, and instructions to enable and
start the service. You can manually start (again as root):

	umask 027
	./ding -loglevel=debug serve -listen localhost:6084 -listenwebhook localhost:6085 -listenadmin localhost:6086 ding.conf

For manual installation instructions, see INSTALL.txt in this repository, or
run "ding help" to print the file included in ding.


# Download

Download a binary for the latest release and latest Go toolchain at:

	https://beta.gobuilds.org/github.com/mjl-/ding@latest/linux-amd64-latest/


# Compiling

Ensure you have a recent Go toolchain and run:

	GOBIN=$PWD CGO_ENABLED=0 go install github.com/mjl-/ding@latest


# Upgrading

If you are upgrading from a version before v0.4.0, you must first upgrade to
v0.3.5, then v0.4.0, and then upgrade further. Versions before v0.4.0 used
PostgreSQL as database. Upgrading to v0.3.5 ensures the PostgreSQL schema is at
its latest. Version v0.4.0 then automatically migrates to the builtin database
storage. For versions after v0.4.0, the "Database" connection field in the
config must be removed.


# Features

- Self-hosted build server. Keep control over your code and builds!
- Simple. Get started quickly, use a simple shell script to make your build. No
  need to learn custom yaml formats to describe how to build.
- Secure, with isolated builds, each build starts under its own unix user id:
  extremely fast, and builds can't interfere with each other. Optionally use
  bubblewrap to isolate the process further.
- (Web) API for all functionality (what the html5/js frontend is using).
- Runs on most unix systems (Linux, BSD's).
- Automatically install Go toolchains when released, and kicking off builds,
  and running build scripts for each supported (or future) Go toolchain
  version.


# Non-features

- Deployments: Different task, different software. Ding exports released files
  which can be picked up by deployment tools.
- Be all things to everybody: Ding does not integrate with every VCS/SCM system
  or deploy system, and does not have a plugin infrastructure. If integrations
  are needed, they happen with shell scripts. This prevents having to implement
  dozens of custom APIs.
- Use docker images: Ding assumes you create self-contained programs, such as
  statically linked Go binaries or JVM .jar files. If you need other services to
  run tests, say a database server, just configure it when setting up your
  repository in ding. If you need certain OS dependencies installed, first try to
  get rid of those dependencies, as a last resort install the dependencies on the
  machine running ding.
- User accounts and access control. A ding instance has a single password, with
  full access.
- Separate servers that run builds. All builds are run on the same machine has
  ding itself.
- Visualized steps for a build. You just write the shell script that gets the
  job done.


# License

Ding is released under an MIT license. See LICENSE.


# FAQ

## Q: Why yet another build server? There are so many already.

Several reasons:
- Some existing tools are too complicated, requiring a big time investment to
  use (plugin systems, own configuration languages). Ding is for developers who
  know how to write a shell script.
- Ding works on different unixes. Many "modern" build servers depend on docker,
  making them Linux-only.
- It is fun creating software like this.

## Q: Does Ding have a website?

Not a separate website, this is it. The ding web app also comes with
documentation.

## Q:  Where is the documentation?

- The README you are reading right now.
- INSTALL.txt with installation instructions, also available with "ding help".
- Documentation behind the "Docs" button in the top-right corner in the Ding web UI.
- API documentation at /ding/ when you've started Ding.

## Q: What does "Ding" mean?

Ding is named after the dent in a surfboard. It needs to be repaired, or the
board will soak up water and become useless or break. Likewise, software needs
to be built regularly, with the latest toolchains, and issues need to be
resolved before the software becomes a lost cause.


# Contact

For feedback, bug reports and questions, please contact mechiel@ueber.net.
