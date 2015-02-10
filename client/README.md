# release.projects.playground/client

Source code for the Vanadium playground web client.

* `build` - Temporary directory used for building the client.
* `bundles` - Default playground examples. Organized as `bundles/<group>/<example>/`.
* _Makefile_ - Targets for building the client (browserifying Javascript, etc.)
* `node_modules` - Disposable directory created by `npm install` - dependency modules.
* _package.json_ - Used by `npm install` to grab playground dependencies.
* `public` - Deployed client website, served by `npm start`.
* `src/javascript` - Scripts implementing the playground client.
* `src/static` - HTML and other static resources for a simple page with a client instance.
* `src/stylesheets` - CSS for playground editor and output.
* _test.sh_ - Script testing correctness of default playground examples.

Requires [npm] and [Node.js].

Build the playground web client:

    make deploy

The command above automatically fetches node dependencies, browserifies Javascript, and
copies all client resources (browserified Javascript, CSS, static files, example bundles)
from `src` and `bundles` to the `public` directory for serving.

Start a server on [localhost:8088](http://localhost:8088):

    npm start

Alternatively, build and start the server in one command with:

    make start

As of dec-2014, the playground doc is [here][playground-doc].

[Node.js]: http://nodejs.org/
[npm]: https://www.npmjs.com/
[playground-doc]: https://docs.google.com/document/d/1OYuE3XLc5CvDKoJSJ2mYjb9wm9IzTttZtP8coJ_t0Wg/edit#heading=h.i9kd9dq3kqco
