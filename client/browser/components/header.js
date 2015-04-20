// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
var debug = require('debug')('components:header:render');
var hg = require('mercury');
var h = require('mercury').h;
var anchor = require('../router/anchor');
var click = require('../event-handlers/click');

module.exports = {
  render: render
};

function render(state, channels) {
  debug('update %o', state);

  return h('header', [
    h('nav.left', [
      anchor({ className: 'logo', href: '/' }, 'Vanadium')
    ]),
    hg.partial(controls, state, channels)
  ]);
}

function controls(state, channels) {
  var bundle = state.uuid ? state.bundles[state.uuid] : null;

  if (! bundle) {
    return h('nav.main');
  } else {
    return h('nav.main', [
      hg.partial(save, bundle, bundle.channels),
      hg.partial(runOrStop, bundle, bundle.channels)
    ]);
  }
}

function save(bundle, channels) {
  return h('a.bundle-save', {
    'href': '#',
    'ev-click': click(channels.save)
  }, 'Save');
}

function runOrStop(bundle, channels) {
  var text;
  var sink;

  if (bundle.running) {
    text = 'Stop';
    sink = channels.stop;
  } else {
    text = 'Run';
    sink = channels.run;
  }

  return h('a.bundle-run-or-stop', {
    className: bundle.running ? 'running' : 'stopped',
    href: '#',
    'ev-click': click(sink)
  }, text);
}
