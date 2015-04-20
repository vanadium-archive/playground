// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
var hg = require('mercury');

module.exports = results;
module.exports.render = require('./render');

function results() {
  var state = hg.state({
    logs: hg.array([]),
    open: hg.value(false),
    follow: hg.value(true),
    debug: hg.value(false),
    channels: {
      follow: follow,
      debug: debug
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

function debug(state, data) {
  var current = state.debug();

  if (data.debug !== current) {
    state.debug.set(data.debug);
  }
}
