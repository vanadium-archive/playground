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

    $ cp ~/.netrc $V23_ROOT/release/projects/playground/go/src/playground/netrc
    $ docker build -t playground $V23_ROOT/release/projects/playground/go/src/playground/.

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

    $ cd $V23_ROOT/release/projects/playground/client && make src/example_bundles
    $ docker run -i playground < $V23_ROOT/release/projects/playground/client/bundles/fortune/bundle_js_go.json


## Running the playground server (compilerd)

Install the playground binaries:

    $ GOPATH=$V23_ROOT/release/projects/playground/go v23 go install playground/...

Run the compiler binary:

    $ $V23_ROOT/release/projects/playground/go/bin/compilerd --listen-timeout=0 --address=localhost:8181

Or, run it without Docker (for faster iterations during development):

    $ PATH=$V23_ROOT/release/go/bin:$V23_ROOT/release/projects/playground/go/bin:$PATH compilerd --listen-timeout=0 --address=localhost:8181 --use-docker=false

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

    MariaDB [(none)]> CREATE DATABASE IF NOT EXISTS playground;

Create a playground user who has access to the playground database:

    MariaDB [(none)]> GRANT ALL PRIVILEGES ON playground.* TO 'playground'@'localhost';

Create config/db.json from example:

    $ cp config/db-local-example.json config/db.json

Edit config/db.json and set username, password, and database.

TODO(ivanpi): Describe cloud storage.

# Running tests

Make sure you have built a docker playground image, following the steps above.

Make sure that MariaDB is installed, following the steps above.

Run sql_test_setup.sh. You will be prompted for your MariaDB password for root
account. You only need to do this once.

    $ ./sql_test_setup.sh

This script will create a playground_test database, and a playground_test user
that can access it.

Run the tests:

    $ GOPATH=$V23_ROOT/release/projects/playground/go v23 go test playground/compilerd/...


# Database migrations

## Running migrations

Build the `sql-migrate` tool:

    $ v23 go install github.com/rubenv/sql-migrate/sql-migrate

Edit config/migrate.yml. Find or define whatever environment you plan to
migrate, and make sure the datasource is correct.

To see the current migration status, run:

    $ $V23_ROOT/third_party/go/bin/sql-migrate status -config=./config/migrate.yml -env=<environment>

To migrate up, first run with -dryrun:

    $ $V23_ROOT/third_party/go/bin/sql-migrate up -config=./config/migrate.yml -env=<environment> -dryrun

If everything looks good, run the tool without -dryrun:

    $ $V23_ROOT/third_party/go/bin/sql-migrate up -config=./config/migrate.yml -env=<environment>

You can undo the last migration with:

    $ $V23_ROOT/third_party/go/bin/sql-migrate down -limit=1 -config=./config/migrate.yml -env=<environment>

For more options and infomation, see https://github.com/rubenv/sql-migrate#usage

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
