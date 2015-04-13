// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var debug = require('debug')('components:results-console:render');
var h = require('mercury').h;
var scroll = require('../../event-handlers/scroll');
var followHook = require('./follow-hook');
var log = require('../log');

module.exports = render;

function render(state, channels) {
  debug('update console %o', state);

  return h('.console', {
    className: state.open ? 'open' : 'closed',
    'ev-scroll': scroll(channels.follow, { scrolling: true }),
    'follow-console': followHook(state.follow)
  }, [
    h('.text', state.logs.map(log.render))
  ]);
}
