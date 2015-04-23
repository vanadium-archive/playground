// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var debug = require('debug')('app');
var domready = require('domready');
var window = require('global/window');
var document = require('global/document');
var hg = require('mercury');
var render = require('./render');
var router = require('./router');
var bundle = require('./components/bundle');
var bundles = require('./components/bundles');
var toast = require('./components/toast');
var error = require('./components/error');
var api = require('./api');

// Make the debug module accessible via the console. This allows the module's
// output to be toggled by opening the developer console and typing:
//
//    debug.enable('*')
//
// It is also possible to only enable specific, namespaced statements. For
// example to only see comments from components.
//
//    debug.enable('component:*')
//
// SEE: https://www.npmjs.com/package/debug
window.debug = require('debug');

domready(function domisready() {
  debug('DOM is ready, initializing mercury app.');

  var state = hg.state({
    bundles: bundles(),
    uuid: hg.value(''),
    toast: toast(),
    title: hg.value('Vanadium Playground'),
    error: hg.value(null)
  });

  router({
    '/#!/': index,
    '/#!/:uuid': show
  }).on('notfound', notfound);

  hg.app(document.body, state, render);

  // Route: "/#!/" - The homepage, show list of available bundles.
  function index(params, route) {
    // Unset state.uuid from previous route.
    state.uuid.set('');

    api.bundles(function(err, list) {
      if (err) {
        state.error.set(error({
          title: 'API Error',
          body: 'There was problem retrieving the list of examples. ' +
            'Please try again later.',
          error: err
        }));

        return;
      }

      var length = list.length;

      for (var i = 0; i < length; i++) {
        var data = list[i];
        state.bundles.put(data.uuid, bundle(data));
      }
    });
  }

  // Route: "/#!/:uuid" - Show a single bundle
  function show(params) {
    state.uuid.set(params.uuid);

    // TODO(jasoncampbell): If there is not an entry for `params.uuid` show a
    // spinner/loader.
    //
    // SEE: https://github.com/vanadium/issues/issues/39

    api.get(params.uuid, function(err, data) {
      if (err) {
        return console.error('TODO: API GET error', err);
      }

      // Set and or update bundle
      state.bundles.put(data.uuid, bundle(data));
    });
  }

  // SEE: https://github.com/vanadium/issues/issues/39
  function notfound(href) {
    console.error('TODO: not found error - %s', href);
  }
});
