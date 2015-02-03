# Building a Docker image and running the playground server locally

## Docker setup

Install Docker:

* Goobuntu: http://go/installdocker
* OS X: https://github.com/boot2docker/osx-installer/releases

On Goobuntu, we recommend overriding the default graph dir (`/var/lib/docker`)
to avoid filling up the root filesystem partition, which is quite small. To do
so, add the following line to your `/etc/default/docker`:

    DOCKER_OPTS="${DOCKER_OPTS} -g /usr/local/google/docker"

Start (or restart) the Docker daemon:

    $ sudo service docker restart

Build the playground Docker image (this will take a while...):

    $ cp ~/.gitcookies $VANADIUM_ROOT/release/projects/playground/go/src/playground/builder/gitcookies
    $ cp ~/.hgrc $VANADIUM_ROOT/release/projects/playground/go/src/playground/builder/hgrc
    $ sudo docker build -t playground $VANADIUM_ROOT/release/projects/playground/go/src/playground/.

Note: If you want to ensure an up-to-date version of Vanadium is installed in
the Docker image, run the above command with the "--no-cache" flag.

The 'docker build' command above will compile builder from the main Vanadium
repository. If you want to use local code instead, open Dockerfile and
uncomment marked lines before running the command.

Test your image (without running compilerd):

    $ cd $VANADIUM_ROOT/release/projects/playground/client && make src/example_bundles
    $ sudo docker run -i playground < $VANADIUM_ROOT/release/projects/playground/client/bundles/fortune/ex0_go/bundle.json

## Running the playground server (compilerd)

Install the playground binaries:

    $ GOPATH=$VANADIUM_ROOT/release/projects/playground/go v23 go install playground/...

Run the compiler binary as root:

    $ sudo $VANADIUM_ROOT/release/projects/playground/go/bin/compilerd --shutdown=false --address=localhost:8181

Or, run it without Docker (for faster iterations during development):

    $ cd $(mktemp -d "/tmp/XXXXXXXX")
    $ PATH=$VANADIUM_ROOT/release/go/bin:$VANADIUM_ROOT/release/projects/playground/go/bin:$PATH compilerd --shutdown=false --address=localhost:8181 --use-docker=false

The server should now be running at http://localhost:8181 and responding to
compile requests at http://localhost:8181/compile.

Add `?pgaddr=//localhost:8181` to any playground page to make the client talk
to your server. Add `?debug=1` to see debug info from the builder.

TODO: storage
