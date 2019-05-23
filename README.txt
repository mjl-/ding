# Ding - simple secure self-hosted build server for developers

Ding builds your software projects from git/mercurial/etc repositories, can run
tests and keep track of released software.

You will typically configure a "post-receive" (web)hook on your git server to
tell Ding to start a build.

Ding will start the compile, run tests, and make resulting binaries/files.
Successful results can be promoted to a release.

All command output of a build is kept around. If builds/tests go wrong, you can
look at the output.

Build dirs are kept around for some time, and garbage collected automatically
when you have more than 10 builds, or after 2 weeks.

Ding provides a web API at /ding/, open it once you've got it installed. It
includes an option to get real-time updates to builds and repositories.

"Ding kick" is a subcommand you can use in a git hook to signal that a build
should start. Github and bitbucket webhooks are also supported.

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
- Runs on unix systems (Linux, BSD's).


# Non-features

Ding does _NOT_ ...

- do deployments: Different task, different software. Ding exports released
files which can be picked up by deployment tools.
- want to be all things to everybody: Ding does not integrate with every
VCS/SCM, does not have a plugin infrastructure, and does not hold your hand.
- use docker images: Ding assumes you create self-contained programs, such as
statically linked Go binaries or JVM .jar files. If you need other services, say
a database server, just configure it when setting up your repository in Ding. If
you need certain OS dependencies installed, first try to get rid of those
dependencies. If that isn't an option, install the dependencies on the build
server.
- call itself "continuous integration" or CI server. Mostly because that term
doesn't seem to be describing what Ding do, it's not integrating anything, it's
just building software.

# License

Ding is released under an MIT license. See LICENSE..


# FAQ

## Q: Why yet another build server? There are so many already.

Several reasons:
- Some existing tools are too complicated. They try to be everything to
everyone. This makes it hard to get started. Hard to do simple things. You have
to invest a lot of time to learn how to use their plugin systems, or their
configuration/scripting languages. Ding is for developers who know how to write
a shell script and don't need more hand-holding.
- This build server works securely on different unixes. Many "modern" build
servers depend on docker, making them Linux-only. Ding also works on BSD's.
- Finally, it's fun creating software like this.

## Q: Does Ding have its own website?

No.

## Q:  Where is the documentation?

- The README you are reading right now.
- INSTALL.txt with installation instructions, also available with "ding help".
- Documentation behind the "Help" button in the top-right corner in Ding.
- API documentation at /ding/ when you've started Ding.

## Q: What does "Ding" mean?

Ding is named after the dent in a surfboard. It needs to be repaired, or the
board will soak up water and become useless or break. Likewise, broken software
builds need to be repaired soon, or the quality of your software goes down.

## Q: I have another question.

That's not a question. But please do send the actual question in.

For feedback, bug reports and questions, please contact mechiel@ueber.net.


# Developing

You obviously need a Go compiler.  But you'll also need to run "make
setup" to install jshint and node-sass through npm (nodejs), to
check the JavaScript code and compile SASS.

Now run: "make build test release"


# Todo

- when sse-stream breaks, don't show it at the top of the page, but in a fixed hovering spot, so you see it even when scrolled down
- when on a build page, show it if a new build is already in progress, with a link to that new build.
- keep track of size of shared homedir. and growth of homedir after a build.
- allow aborting a build
- write test code
- improve showing the cause of a failed build. 1. show then just last single line of output (make just prints that it failed at the end). 2. create files in output/ earlier? so we don't show errors about missing such files when the vcs clone failes (eg due to no git in path, or no permision to run build.sh (eg because a dir leading to build.sh isn't accessible).
- add prometheus metrics for builds. how long they take, if they succeed, etc.
- add button to queue a build, low prio. then only run one low-prio job at a time. for rebuilds of all repo's after a toolchain update.
- rewrite ui in typescript with tuit
- when cloning, clone from previous checkout, then pull changes from remote as need, should be faster, especially for larger repo's.
- attempt to detect pushes of commits+tag, don't build for both the commit and the tag if they're about the same thing.
- allow configuring a cleanup script, that is run when a builddir is removed. eg for dropping a database that was created in build.sh.
- think about adding support for updating toolchains, like go get golang.org/dl/go1.2.3 && go1.2.3 download
- add authentication to application. need to figure out how to keep a dashboard. and how to do auth on /events
- more ways to send out notifications? eg webhook, telegram, slack.
