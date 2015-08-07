# release.projects.playground/client

Source code for the Vanadium playground web client.

As of dec-2014, the playground doc is [here][playground-doc].

## Directory structure

* `bin` - Utility scripts for building the client.
* `browser` - JS modules to be compiled/bundled via `[browserify]`.
* _Makefile_ - Targets for building the client (browserifying Javascript, compiling CSS, etc.).
* `node_modules` - JS dependencies created by `npm install`.
* _package.json_ - Used by `npm install` to resolve and download dependencies.
* `public` - Static assets, including JS/CSS bundles, served by `make start`.
* `stylesheets` - CSS to be compiled to `public/bundle.css`.
* `test` - Client TAP unit tests.

Requires [npm] and [Node.js].

Build the playground web client:

    make

The command above automatically fetches Node dependencies and builds necessary
assets for the client bundles to the `public` directory for serving.

## Local server

Start a server on http://127.0.0.1:8088

    make start

This command will also install any dependencies and build the required
bundles.

To run the local dev server with a different configuration you can use
variables:

    host=`hostname` port=4080 make start

The client running on localhost needs to have the backend address configured
via the `pgaddr=` query parameter.

## public/bundle.{js,css}

The default make task will build these for you. The `Makefile` already has
globbing targets for bundling JS and CSS in the `browser` and `stylesheets`
directories. See these directories for entry points (index files). Such files
are bundled into single `public/bundle.*` files.

The CSS/JS bundles do not rebuild automatically via the server. If you would
like the effect of automatic building and seeing your changes on browser
refresh you can use the `watch` utility.

    watch make

# Deploy

If you do not have access to the vanadium-staging GCE account ping
jasoncampbell@. Once you have access you will need to login to the account via
the command line.

    gcloud auth login

To deploy the playground client to https://staging.playground.v.io use the
make target `deploy-staging`.

    make deploy-staging

[Node.js]: http://nodejs.org/
[npm]: https://www.npmjs.com/
[playground-doc]: https://docs.google.com/document/d/1OYuE3XLc5CvDKoJSJ2mYjb9wm9IzTttZtP8coJ_t0Wg/edit#heading=h.i9kd9dq3kqco
[infrastructure]: https://vanadium.googlesource.com/infrastructure/+/master/nginx/README.md
[browserify]: https://www.npmjs.com/package/browserify
