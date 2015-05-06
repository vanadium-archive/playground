// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var debug = require('debug')('components:bundle:state');
var hg = require('mercury');
var log = require('../log');
var api = require('../../api');
var results = require('../results');
var router = require('../../router');
var toArray = require('../../util').toArray;

module.exports = bundle;
module.exports.render = require('./render');

function bundle(json) {
  var state = hg.state({
    uuid: hg.value(json.uuid),
    files: hg.varhash({}),
    tab: hg.value(json.files[0].name),
    running: hg.value(false),
    reset: hg.value(false),
    results: results(),
    channels: {
      tab: tab,
      run: run,
      stop: stop,
      save: save,
      fileChange: fileChange,
      showResults: showResults
    }
  });

  // Loop thorugh the files array in the `json` object and update the
  // `state.files` varhash.
  var length = json.files.length;

  for (var i = 0; i < length; i++) {
    var file = json.files[i];
    // NOTE: hg.varhash has a bug where certain keys such as "delete" can not
    // be used ( https://github.com/nrw/observ-varhash/issues/2 ), once we
    // start allowing user's to create thier own files we will need to
    // accomodate the possibility of running into naming collisions for
    // "name", "get", "put", "delete" by possibily normalizing the keys with a
    // prefix.
    state.files.put(file.name, file);
  }

  // When the console is running update the resultsConsole state.
  state.running(function update(running) {
    debug('run changed: %s', running);

    // If running clear previous logs and open the console.
    if (running) {
      state.results.open.set(true);
      state.results.logs.set(hg.array([]));
      state.results.follow.set(true);
    }
  });

  return state;
}

function showResults(state, data) {
  state.results.open.set(true);
}

// When a file's contents change via the editor update the state.
function fileChange(state, data) {
  var current = state.files.get(data.name);

  if (current.body !== data.body) {
    state.files.put(data.name, data);
  }
}

// Change the current tab.
function tab(state, data) {
  state.tab.set(data.tab);
}

// "stop" the code run. This will hide the console...
function stop(state, data) {
  state.running.set(false);
  // TODO(jasoncampbell): stop pending xhr.
}

// Doesn't "save" in a normal sense as the request generates a new resource.
// This is more like a "Fork" but currently the only thing we have that might
// resemble saving.
function save(state) {
  var data = {
    files: toArray(state.files())
  };

  api.create(data, function(err, data) {
    if (err) {
      // TODO(jasoncampbell): handle error appropriately.
      //
      // SEE: https://github.com/vanadium/issues/issues/39
      throw err;
    }

    // Since a new id is generated and the id is in the url the router needs
    // to be updated.
    //
    // TODO(jasoncampbell): put the the value of the new bundle here, maybe
    // don't trigger a reload.
    router.href.set(data.uuid);
  });
}

// Run the code remotely.
function run(state) {
  debug('running');
  state.running.set(true);

  // Temporary message to provide feedback and show that the run is happening.
  state.results.logs.push(log({
    File: 'web client',
    Message: 'Run request initiated.',
    Stream: 'stdout'
  }));

  var data = {
    uuid: state.uuid(),
    files: toArray(state.files())
  };

  var stream = api.run(data);

  stream.on('error', function(err) {
    // TODO(jasoncampbell): handle errors appropriately.
    throw err;
  });

  stream.on('data', function ondata(data) {
    state.results.logs.push(log(data));
  });

  stream.on('end', function() {
    state.running.set(false);
  });
}
