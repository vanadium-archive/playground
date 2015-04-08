FROM ubuntu
RUN /usr/sbin/useradd -d /home/playground -m playground

# Install various prereqs.
RUN apt-get update
RUN apt-get install -y curl git nodejs npm
# node -> nodejs needed for Ubuntu.
RUN ln -s "$(which nodejs)" "$(dirname $(which nodejs))/node"

# Install Go. Note, the apt-get "golang" target is too old.
RUN (cd /tmp; curl -O https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz)
RUN tar -C /usr/local -xzf /tmp/go1.4.2.linux-amd64.tar.gz
ENV PATH /usr/local/go/bin:$PATH

ENV HOME /root
ENV V23_ROOT /usr/local/vanadium
ENV GOPATH $V23_ROOT/release/projects/playground/go
ENV VDLPATH $GOPATH
ENV PATH $V23_ROOT/release/projects/playground/go/bin:$V23_ROOT/release/go/bin:$V23_ROOT/bin:$PATH

# Setup Vanadium.
# Note: This will be cached! If you want to re-build the docker image using
# fresh Vanadium code, you must pass "--no-cache" to the docker build command.
# See README.md.
ADD netrc /root/.netrc
RUN curl -u vanadium:D6HT]P,LrJ7e https://v.io/bootstrap | bash
RUN rm /root/.netrc

# Install the release/javascript/core library.
# TODO(nlacasse): Switch to "npm install -g vanadium" once release/javascript/core
# is publicly visible in NPM.
# NOTE(sadovsky): NPM is flaky. If any of the NPM commands below fail, simply
# retry them.
WORKDIR /home/playground
RUN npm install --production $V23_ROOT/release/javascript/core

# Install Vanadium libraries and playground binaries.
RUN v23 go install -v v.io/...
RUN v23 go install -v playground/...

# Uncomment the following lines to install a version of the builder tool using
# your local version of the code. This is useful when developing and testing
# local changes.
#RUN rm -f $V23_ROOT/release/projects/playground/go/bin/builder
#RUN rm -rf $V23_ROOT/release/projects/playground/go/src/playground/Makefile
#RUN rm -rf $V23_ROOT/release/projects/playground/go/src/playground/builder/
#RUN rm -rf $V23_ROOT/release/projects/playground/go/src/playground/lib/
#ADD Makefile $V23_ROOT/release/projects/playground/go/src/playground/Makefile
#ADD builder/ $V23_ROOT/release/projects/playground/go/src/playground/builder/
#ADD lib/ $V23_ROOT/release/projects/playground/go/src/playground/lib/
#RUN make builder

USER playground
WORKDIR /home/playground
# exec form - passes through command-line arguments and signals.
ENTRYPOINT ["builder"]
