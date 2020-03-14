# Ding - simple secure self-hosted build server for developers

Ding lets you configure repositories and build scripts and compile your
software. For a build, ding:
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
start. Github and bitbucket webhooks are also supported.

See INSTALL.txt for installation instructions.


# Requirements

- PostgreSQL database
- BSD/Linux machine
- git/mercurial or any other version control software you want to use

Ding is distributed as a self-contained binary. It includes installation
instructions (run "ding help") and database setup/upgrade scripts ("ding
upgrade").


# Download

Get the latest version at https://github.com/mjl-/ding/releases/latest.


# Features

- Self-hosted build server. Keep control over your code and builds!
- Simple. Get started quickly, experience the power of simplicity, use your
  existing skills, avoid the burden of complex systems.
- Secure, with isolated builds, each build starts under its own unix user id:
  extremely fast, and builds can't interfere with each other.
- (Web) API for all functionality (what the html5/js frontend is using).
- Runs on most unix systems (Linux, BSD's).


# Non-features

- Deployments: Different task, different software. Ding exports released files
  which can be picked up by deployment tools.
- Be all things to everybody: Ding does not integrate with every VCS/SCM, does
  not have a plugin infrastructure.
- Use docker images: Ding assumes you create self-contained programs, such as
  statically linked Go binaries or JVM .jar files. If you need other services to
  run tests, say a database server, just configure it when setting up your
  repository in Ding. If you need certain OS dependencies installed, first try to
  get rid of those dependencies, as a last resort install the dependencies on the
  machine running ding.


# License

Ding is released under an MIT license. See LICENSE..


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

No.

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

For feedback, bug reports and questions, please contact mechiel@ueber.net.


# Developing

You need a Go compiler and nodejs+npm with jshint and node-sass.  Run "make
setup" to install the js dependencies.

Now run: "make build test"


# Todo

- write test code
- on reconnect after sse failure, make sure our state is up to date again. it isn't now.
- improve showing the cause of a failed build. 1. show then just last single line of output (make just prints that it failed at the end). 2. create files in output/ earlier? so we don't show errors about missing such files when the vcs clone failes (eg due to no git in path, or no permision to run build.sh (eg because a dir leading to build.sh isn't accessible).
- rewrite ui in typescript with tuit
- when cloning, clone from previous checkout, then pull changes from remote as need, should be faster, especially for larger repo's.
- attempt to detect pushes of commits+tag, don't build for both the commit and the tag if they're about the same thing.
- allow configuring a cleanup script, that is run when a builddir is removed. eg for dropping a database that was created in build.sh.
- think about adding support for updating toolchains, like go get golang.org/dl/go1.2.3 && go1.2.3 download
- add authentication to the web ui. need to figure out how to keep a dashboard. and how to do auth on /events. and if this is really worthwhile.
- parse & process the output of a build as it comes in, instead of when the build is done. allows making release-files earlier in the process, eg before slow tests are run.
