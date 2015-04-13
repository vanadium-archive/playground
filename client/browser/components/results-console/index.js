// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var hg = require('mercury');
var debug = require('debug')('components:results-console:state');

module.exports = resultsConsole;
module.exports.render = require('./render');

function resultsConsole() {
  debug('create');

  var state = hg.state({
    logs: hg.array([]),
    open: hg.value(false),
    follow: hg.value(true),
    channels: {
      follow: follow
    }
  });

  return state;
}

function follow(state, data) {
  var following = state.follow();

  if (data.scrolling && following) {
    state.follow.set(false);
  }

  if (data.scrolledToBottom) {
    state.follow.set(true);
  }
}
