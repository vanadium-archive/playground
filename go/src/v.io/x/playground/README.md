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

    $ cp ~/.netrc $JIRI_ROOT/release/projects/playground/go/src/v.io/x/playground/netrc
    $ docker build -t playground $JIRI_ROOT/release/projects/playground/go/src/v.io/x/playground/.

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

    $ $JIRI_ROOT/release/projects/playground/go/bin/pgadmin bundle make fortune js-go | docker run -i playground


## Running the playground server (compilerd)

Install the playground binaries:

    $ GOPATH=$JIRI_ROOT/release/projects/playground/go v23 go install v.io/x/playground/...

Run the compilerd binary:

    $ $JIRI_ROOT/release/projects/playground/go/bin/compilerd --listen-timeout=0 --address=localhost:8181 --origin='*'

Or, run it without Docker (for faster iterations during development):

    $ PATH=$JIRI_ROOT/release/go/bin:$JIRI_ROOT/release/projects/playground/go/bin:$PATH compilerd --listen-timeout=0 --address=localhost:8181 --origin='*' --use-docker=false

The server should now be running at http://localhost:8181 and responding to
compile requests at http://localhost:8181/compile.

Add `?pgaddr=http://localhost:8181` to any playground page on localhost to
make the client talk to your server. Add `?debug=1` to see debug info from
the builder.

## Running local SQL database

NOTE: These instructions should only be used for local development and testing,
and not for production deploys.

Install MariaDB:

    $ sudo apt-get install mariadb-server

Sign in to Maria as root:

    $ mysql -u root -p

Create playground databases:

    MariaDB [(none)]> CREATE DATABASE IF NOT EXISTS pg_moria;

Create a playground user who has access to the playground database:

    MariaDB [(none)]> GRANT ALL PRIVILEGES ON pg_moria.* TO 'pg_gandalf'@'localhost' IDENTIFIED BY 'mellon';

Create `config/db.json` from default:

    $ cp config/db-local-default.json config/db.json

Alternatively, make your own from example.

The compilerd server can now be started with persistence enabled by:

    $ make start


# Running tests

Make sure you have built a docker playground image, following the steps above.

Make sure that MariaDB is installed, following the steps above.

Run sql_test_setup.sh. You will be prompted for your MariaDB password for root
account. You only need to do this once.

    $ ./sql_test_setup.sh

This script will create a playground_test database, and a playground_test user
that can access it.

Run the tests:

    $ GOPATH=$JIRI_ROOT/release/projects/playground/go v23 go test v.io/x/playground/compilerd/...


# Database migrations

## Running migrations

Migrations use the `github.com/rubenv/sql-migrate` library, wrapped in a tool
`pgadmin` to allow TLS connections.

Create the database and `config/db.json` file following instructions above.

To migrate up, first run with -n (dry run):

    $ $JIRI_ROOT/release/projects/playground/go/bin/pgadmin -sqlconf=./config/db.json migrate up -n

If everything looks good, run the same command without -n; alternatively, run:

    $ make updatedb

You can undo the last migration with:

    $ $JIRI_ROOT/release/projects/playground/go/bin/pgadmin -sqlconf=./config/db.json migrate down -limit=1

For more options and infomation, run:

    $ $JIRI_ROOT/release/projects/playground/go/bin/pgadmin help

and see https://github.com/rubenv/sql-migrate

## Writing migrations

Migrations are kept in the `migrations` directory. They are ordered
alphabetically, so please name each migration consecutively. Never delete or
modify an existing migration. Only add new ones.

Each migration file must define an "up" section, which begins with the comment

    -- +migrate Up

and a "down" section, which begins with the comment

    -- +migrate Down

Applying a single migration "up" and then "down" should leave the database in
the same state it was to begin with.

For more information on writing migrations, see https://github.com/rubenv/sql-migrate#writing-migrations

## Bootstrapping default examples

The playground client expects to find up-to-date default examples already
present in the database to use as templates for editing. The unpacked example
source code can be found in the `bundles` directory, described in
`bundles/config.json`. Each example is obtained by filtering files from a
folder according to a glob-like configuration and bundling them into a JSON
object that the client can parse.

Bundling and loading the examples into a fresh database, as well as updating,
is handled by the `pgadmin` tool:

    $ $JIRI_ROOT/release/projects/playground/go/bin/pgadmin -sqlconf=./config/db.json bundle bootstrap

Or simply:

    $ make bootstrap

When adding new default examples or implementations of existing ones,
`bundles/config.json` must also be edited to include them in bootstrapping and
tests. For config file format documentation, see:

    $ $JIRI_ROOT/release/projects/playground/go/bin/pgadmin bundle help bootstrap
