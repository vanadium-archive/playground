// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var hg = require('mercury');
var assert = require('assert');

module.exports = error;
module.exports.render = require('./render');

// A temporary way to display errors meaningfully to the user while logging
// relevant bits to the developer console.
function error(data) {
  assert.ok(data.error, 'data.error is required');
  assert.ok(data.error instanceof Error, 'data.error must be an error');

  var state = hg.state({
    title: data.title || data.error.name,
    body: data.body || data.error.message,
    error: data.error
  });

  console.error('Error reported via components/error');
  console.error(data.error.stack);

  return state;
}
