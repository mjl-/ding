# Ding - simple secure self-hosted private build server for developers

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

Non-released builds older than 2 weeks or beyond the 10th build are
automatically cleaned up.

Command "ding kick" can be used in a git hook to signal that a build should
start. Gitea, github and bitbucket webhooks are also supported.

Command "ding build" can be used locally to run a build script similar to how
it would run on the build server. This allows testing build scripts without
pushing any commits. It clones the git or hg repository in the current working
directory, sets up environment variables and destination build directories, and
calls the provided build script. Optionally isolated using bubblewrap (bwrap).

Run "ding quickstart" to generate a config file, and a systemd service file
when on linux.  See INSTALL.txt for manual installation instructions.


# Requirements

- BSD/Linux machine
- git/mercurial or any other version control software you want to use

Ding is distributed as a self-contained binary. It includes installation
instructions (run "ding help").


# Download

Download a binary at:

	https://beta.gobuilds.org/github.com/mjl-/ding@latest


# Compiling

Run:

	CGO_ENABLED=0 go install github.com/mjl-/ding@latest


# Upgrading

If you are upgrading from a version before v0.4.0, you must first upgrade to
v0.3.5, then v0.4.0, and then upgrade further. Versions before v0.4.0 used
PostgreSQL as database. Upgrading to v0.3.5 ensures the PostgreSQL schema is at
its latest. Version v0.4.0 then automatically migrates to the builtin database
storage. For versions after v0.4.0, the "Database" connection field in the
config should be removed.


# Features

- Self-hosted build server. Keep control over your code and builds!
- Simple. Get started quickly, experience the power of simplicity, use your
  existing skills, avoid the burden of complex systems.
- Secure, with isolated builds, each build starts under its own unix user id:
  extremely fast, and builds can't interfere with each other.
- (Web) API for all functionality (what the html5/js frontend is using).
- Runs on most unix systems (Linux, BSD's).
- Automatically install Go toolchains when released, and kicking off builds.


# Non-features

- Deployments: Different task, different software. Ding exports released files
  which can be picked up by deployment tools.
- Be all things to everybody: Ding does not integrate with every VCS/SCM, does
  not have a plugin infrastructure.
- Use docker images: Ding assumes you create self-contained programs, such as
  statically linked Go binaries or JVM .jar files. If you need other services to
  run tests, say a database server, just configure it when setting up your
  repository in ding. If you need certain OS dependencies installed, first try to
  get rid of those dependencies, as a last resort install the dependencies on the
  machine running ding.


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

No, this is the website.

## Q:  Where is the documentation?

- The README you are reading right now.
- INSTALL.txt with installation instructions, also available with "ding help".
- Documentation behind the "Help" button in the top-right corner in the Ding web UI.
- API documentation at /ding/ when you've started Ding.

## Q: What does "Ding" mean?

Ding is named after the dent in a surfboard. It needs to be repaired, or the
board will soak up water and become useless or break. Likewise, software needs
to be built regularly, with the latest toolchains, and issues need to be
resolved before the software becomes a lost cause.


# Contact

For feedback, bug reports and questions, please contact mechiel@ueber.net.


# Developing

You need a Go compiler and nodejs+npm with jshint and sass.  Run "make
setup" to install the js dependencies.

Now run: "make build test"


# Todo

- when doing a concurrent build, check how much memory is available, and how much the build likely needs (based on previous build, need to start keeping track of rusage), and delay execution if there isn't enough memory.
- authentication on downloadable files? currently very useful to just wget a built binary (with internal endpoints).
- on reconnect after sse failure, make sure our state is up to date again. it isn't now.

- parse & process the output of a build as it comes in, instead of when the build is done. allows making result-files earlier in the process, eg before slow tests are run.
- when cloning, clone from previous checkout, then pull changes from remote as need, should be faster, especially for larger repo's.
- attempt to detect pushes of commits+tag, don't build for both the commit and the tag if they're about the same thing.
- allow configuring a cleanup script, that is run when a builddir is removed. eg for dropping a database that was created in build.sh.
