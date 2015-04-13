// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var debug = require('debug')('components:bundle:render');
var anchor = require('../../router/anchor');
var h = require('mercury').h;
var hg = require('mercury');
var results = require('../results-console');
var toArray = require('../../util').toArray;

module.exports = render;

function render(state) {
  debug('update %o', state);

  if (! state) {
    return h('.bundle', [
      h('p', 'Loading...')
    ]);
  }

  return h('.bundle', [
    h('.pg', [
      hg.partial(anchorbar, state, state.channels),
      hg.partial(tabs, state, state.channels),
      hg.partial(controls, state, state.channels),
      hg.partial(editors, state, state.channels),
      hg.partial(results.render,
        state.resultsConsole,
        state.resultsConsole.channels)
    ])
  ]);
}

var options = { preventDefault: true };

function anchorbar(state, channels) {
  return h('.widget-bar', [
    h('p', [
      h('strong', 'anchor:'),
      anchor({ href: '/' + state.uuid }, state.uuid)
    ])
  ]);
}

function tabs(state, channels) {
  var files = toArray(state.files);

  return h('span.tabs', files.map(tab, state));
}

function tab(file, index, array) {
  var state = this;
  var channels = state.channels;

  return h('span.tab', {
    className: state.tab === file.name ? 'active' : '',
    'ev-click': hg.sendClick(channels.tab, { tab: file.name }, options)
  }, file.name);
}

function controls(state, channels) {
  return h('span.btns', [
    hg.partial(runButton, state, channels),
    h('button.btn', {
      'ev-click': hg.sendClick(channels.save)
    }, 'Save')
  ]);
}

function runButton(state, channels) {
  var text = 'Run Code';
  var sink = channels.run;

  if (state.running) {
    text = 'Stop';
    sink = channels.stop;
  }

  return h('button.btn', {
    'ev-click': hg.sendClick(sink)
  }, text);
}

// TODO(jasoncampbell): It makes sense to break the editor into it's own
// component as we will be adding features...
var aceWidget = require('../../widgets/ace-widget');
var aceChange = require('../../event-handlers/ace-change');

function editors(state, channels) {
  var files = toArray(state.files);

  return h('.editors', files.map(function(file) {
    return h('.editor', {
      className: (state.tab === file.name ? 'active' : ''),
      'ev-ace-change': aceChange(state.channels.fileChange),
    }, [
      aceWidget(file)
    ]);
  }));
}
