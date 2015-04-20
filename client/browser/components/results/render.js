// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
var h = require('mercury').h;
var hg = require('mercury');
var scroll = require('../../event-handlers/scroll');
var click = require('../../event-handlers/click');
var followHook = require('./follow-hook');
var log = require('../log');
var format = require('format');

module.exports = render;

function render(state, channels) {
  return h('.results', [
    hg.partial(controls, state, channels),
    h('.console', {
      className: state.debug ? 'debug' : ''
    }, [
      h('.scroller', {
        'ev-scroll': scroll(channels.follow, { scrolling: true }),
        'follow-console': followHook(state.follow)
      }, state.logs.map(log.render))
    ])
  ]);
}

function controls(state, channels) {
  var onOrOff = (state.debug ? 'on' : 'off');
  var title = format('Toggle debug console output %s.', onOrOff);
  var text = format(' Debug: %s', onOrOff);

  return h('.controls', [
    'Results',
    h('a.debug', {
      href: '#',
      'ev-click': click(channels.debug, { debug: ! state.debug }),
      title: title
    }, text)
  ]);
}
