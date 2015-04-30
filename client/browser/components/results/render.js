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
var debug = require('debug')('components:results:render');

module.exports = render;

function render(state, channels) {
  debug('update %o', state);

  channels = channels || state.channels;

  return h('.results', {
    className: state.open ? 'opened' : 'closed'
  },
  [
    hg.partial(controls, state, channels),
    hg.partial(terminal, state, channels)
  ]);
}

function controls(state, channels) {
  var onOrOff = (state.debug ? 'on' : 'off');
  var title = format('Toggle debug console output %s.', onOrOff);
  var text = format(' Debug: %s', onOrOff);

  return h('.results-controls', [
    h('a.toggle-display', {
      href: '#',
      title: (state.open ? 'Close' : 'Open') + ' the results console.',
      'ev-click': click(channels.toggle),
    }),
    h('.title', 'Results'),
    h('a.debug-button', {
      href: '#',
      'ev-click': click(channels.debug, { debug: ! state.debug }),
      title: title
    }, text),
  ]);
}

function terminal(state, channels) {
  return h('.results-console', {
    className: state.debug ? 'debug' : ''
  }, [
    h('.scroller', {
      'ev-scroll': scroll(channels.follow, { scrolling: true }),
      'follow-console': followHook(state.follow)
    }, state.logs.map(log.render))
  ]);
}
