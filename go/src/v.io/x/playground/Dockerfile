FROM ubuntu
RUN /usr/sbin/useradd -d /home/playground -m playground

# Install various prereqs.
RUN apt-get update && \
    apt-get install -y curl git make && \
    apt-get clean

# Install Go. Note, the apt-get "golang" target is too old.
ENV GOLANG_TAR go1.5.3.linux-amd64.tar.gz
RUN (cd /tmp; curl -O https://storage.googleapis.com/golang/${GOLANG_TAR}) && \
    tar -C /usr/local -xzf /tmp/${GOLANG_TAR} && \
    rm /tmp/${GOLANG_TAR}
ENV PATH /usr/local/go/bin:$PATH

ENV HOME /root
ENV JIRI_ROOT /usr/local/vanadium
ENV GOPATH $JIRI_ROOT/release/projects/playground/go
ENV VDLPATH $JIRI_ROOT/release/projects/playground/go/src
ENV PATH $JIRI_ROOT/release/projects/playground/go/bin:$JIRI_ROOT/.jiri_root/bin:$JIRI_ROOT/release/go/bin:$PATH

# Setup Vanadium.
# Note: This will be cached! If you want to re-build the docker image using
# fresh Vanadium code, you must pass "--no-cache" to the docker build command.
# See README.md.
RUN curl -f https://vanadium.github.io/bootstrap.sh | bash

# Setup Vanadium profiles.
RUN jiri profile-v23 install v23:base

# Install Vanadium libraries and playground binaries.
RUN jiri go install -v v.io/...

# Uncomment the following lines to install a version of the builder tool using
# your local version of the code. This is useful when developing and testing
# local changes.
#RUN rm -f $JIRI_ROOT/release/projects/playground/go/bin/builder
#RUN rm -rf $JIRI_ROOT/release/projects/playground/go/src/playground/Makefile
#RUN rm -rf $JIRI_ROOT/release/projects/playground/go/src/playground/builder/
#RUN rm -rf $JIRI_ROOT/release/projects/playground/go/src/playground/lib/
#ADD Makefile $JIRI_ROOT/release/projects/playground/go/src/playground/Makefile
#ADD builder/ $JIRI_ROOT/release/projects/playground/go/src/playground/builder/
#ADD lib/ $JIRI_ROOT/release/projects/playground/go/src/playground/lib/
#RUN make builder

USER playground
WORKDIR /home/playground
# exec form - passes through command-line arguments and signals.
ENTRYPOINT ["builder"]
