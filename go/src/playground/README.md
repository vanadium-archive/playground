# Building a Docker image and running the playground server locally

## Docker setup

Install Docker:

* Goobuntu: http://go/installdocker
* OS X: https://github.com/boot2docker/osx-installer/releases

On Goobuntu, we recommend overriding the default graph dir (`/var/lib/docker`)
to avoid filling up the root filesystem partition, which is quite small. To do
so, add the following line to your `/etc/default/docker`:

    DOCKER_OPTS="${DOCKER_OPTS} -g /usr/local/google/docker"

Add your user to the docker group:

    $ sudo usermod -a -G docker $(whoami)

Start (or restart) the Docker daemon:

    $ sudo service docker restart

Build the playground Docker image (this will take a while...):

    $ cp ~/.netrc $VANADIUM_ROOT/release/projects/playground/go/src/playground/netrc
    $ docker build -t playground $VANADIUM_ROOT/release/projects/playground/go/src/playground/.

Note: If you want to ensure an up-to-date version of Vanadium is installed in
the Docker image, run the above command with the "--no-cache" flag.

Note: If you have only a .gitcookies googlesource.com entry and not a .netrc
one, you can convert it to a .netrc entry using:

    $ cat ~/.gitcookies | grep vanadium.googlesource.com | tail -n 1 | sed -E 's/(\S+)\s+(\S+\s+){5}([^=]+)=(\S+)/machine \1 login \3 password \4/' >> ~/.netrc

(see http://www.chromium.org/chromium-os/developer-guide/gerrit-guide for details)

The 'docker build' command above will compile builder from the main Vanadium
repository. If you want to use local code instead, open Dockerfile and
uncomment marked lines before running the command.

Test your image (without running compilerd):

    $ cd $VANADIUM_ROOT/release/projects/playground/client && make src/example_bundles
    $ docker run -i playground < $VANADIUM_ROOT/release/projects/playground/client/bundles/fortune/bundle_js_go.json

## Running the playground server (compilerd)

Install the playground binaries:

    $ GOPATH=$VANADIUM_ROOT/release/projects/playground/go v23 go install playground/...

Run the compiler binary:

    $ $VANADIUM_ROOT/release/projects/playground/go/bin/compilerd --listen-timeout=0 --address=localhost:8181

Or, run it without Docker (for faster iterations during development):

    $ PATH=$VANADIUM_ROOT/release/go/bin:$VANADIUM_ROOT/release/projects/playground/go/bin:$PATH compilerd --listen-timeout=0 --address=localhost:8181 --use-docker=false

The server should now be running at http://localhost:8181 and responding to
compile requests at http://localhost:8181/compile.

Add `?pgaddr=http://localhost:8181` to any playground page on localhost to
make the client talk to your server. Add `?debug=1` to see debug info from
the builder.

TODO(ivanpi): Describe storage.
