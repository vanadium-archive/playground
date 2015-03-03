FROM ubuntu
RUN /usr/sbin/useradd -d /home/playground -m playground

# Install various prereqs.
RUN apt-get update
RUN apt-get install -y curl g++ git libc6-i386 make mercurial python

# Install Go. Note, the apt-get "golang" target is too old.
RUN (cd /tmp; curl -O https://storage.googleapis.com/golang/go1.4.linux-amd64.tar.gz)
RUN tar -C /usr/local -xzf /tmp/go1.4.linux-amd64.tar.gz
ENV PATH /usr/local/go/bin:$PATH

ENV HOME /root
ENV VANADIUM_ROOT /usr/local/vanadium
ENV GOPATH /home/playground:$VANADIUM_ROOT/release/go:$VANADIUM_ROOT/release/projects/playground/go
ENV VDLPATH $GOPATH

# Setup Vanadium and Vanadium profiles.
# Note: This will be cached! If you want to re-build the docker image using
# fresh Vanadium code, you must pass "--no-cache" to the docker build command.
# See README.md.
ADD deploy/netrc /root/.netrc
RUN curl -u vanadium:D6HT]P,LrJ7e https://dev.v.io/noproxy/vanadium-setup.sh | bash
RUN rm /root/.netrc
ADD deploy/hgrc /root/.hgrc
RUN $VANADIUM_ROOT/bin/v23 profile setup web
RUN rm /root/.hgrc

# Install the release/javascript/core library.
# TODO(nlacasse): Switch to "npm install -g vanadium" once release/javascript/core
# is publicly visible in NPM.
# NOTE(sadovsky): NPM is flaky. If any of the NPM commands below fail, simply
# retry them.
WORKDIR /usr/local/vanadium/release/javascript/core
RUN $VANADIUM_ROOT/environment/cout/node/bin/npm install --production
RUN $VANADIUM_ROOT/environment/cout/node/bin/npm link
WORKDIR /home/playground
RUN $VANADIUM_ROOT/environment/cout/node/bin/npm link vanadium

# Install Vanadium Go dependencies.
WORKDIR /usr/local/vanadium/release
ENV PATH $VANADIUM_ROOT/release/go/bin:$VANADIUM_ROOT/release/projects/playground/go/bin:$VANADIUM_ROOT/bin:$PATH
RUN v23 go install v.io/x/ref/...
RUN v23 go install playground/...

# Uncomment the following lines to install a version of the builder tool using
# your local version of the code. This is useful when developing and testing
# local changes.
#RUN rm -f $VANADIUM_ROOT/release/projects/playground/go/bin/builder
#RUN rm -rf $VANADIUM_ROOT/release/projects/playground/go/src/playground/builder/
#RUN rm -rf $VANADIUM_ROOT/release/projects/playground/go/src/playground/lib/
#ADD builder/ $VANADIUM_ROOT/release/projects/playground/go/src/playground/builder/
#ADD lib/ $VANADIUM_ROOT/release/projects/playground/go/src/playground/lib/
#RUN v23 go install playground/builder/...

# Copy proxyd's main.go to ./proxyd_main.go, then uncomment the following
# lines to install a version of proxyd (used by the builder tool) using your
# local version of the code. This is useful when developing and testing local
# changes.
#RUN rm -f $VANADIUM_ROOT/release/go/bin/proxyd
#ADD proxyd_main.go $VANADIUM_ROOT/release/go/src/v.io/x/ref/services/proxy/proxyd/main.go
#RUN v23 go install v.io/x/ref/services/proxy/proxyd

USER playground
WORKDIR /home/playground
ENTRYPOINT /usr/local/vanadium/release/projects/playground/go/bin/builder
