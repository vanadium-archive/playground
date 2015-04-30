// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var debug = require('debug')('components:bundle:render');
var h = require('mercury').h;
var hg = require('mercury');
var results = require('../results').render;
var toArray = require('../../util').toArray;
var click = require('../../event-handlers/click');
var path = require('path');
var aceWidget = require('../../widgets/ace-widget');
var aceChange = require('../../event-handlers/ace-change');

module.exports = render;

function render(state) {
  debug('update %o', state);

  // NOTE: It is possible to try and render a bundle without it being
  // populated from the API yet. In that case show a loader.
  if (! state) {
    return h('.bundle', [
      h('p.loading', 'Loading...')
    ]);
  }

  return h('.bundle', [
    hg.partial(code, state, state.channels),
    hg.partial(results, state.results, state.results.channels),
  ]);
}

function code(state, channels) {
  return h('.code', [
    hg.partial(tabs, state, channels),
    hg.partial(editors, state, channels)
  ]);
}

function tabs(state, channels) {
  var files = toArray(state.files);
  var children = files.map(tab, state);

  // .spacer for flex box pushing a.show-results to the far right
  children.push(h('.spacer'));

  children.push(h('a.show-results', {
    title: 'Open the results console.',
    href: '#',
    'ev-click': click(channels.showResults)
  }));

  return h('.tabs', children);
}

function tab(file, index, array) {
  var state = this;
  var channels = state.channels;
  var name = path.basename(file.name);

  return h('a.tab', {
    className: state.tab === file.name ? 'active' : '',
    href: '#',
    'ev-click': click(channels.tab, { tab: file.name })
  }, name);
}



function editors(state, channels) {
  var files = toArray(state.files);

  return h('.editors', files.map(editor, state));
}

function editor(file, index, files) {
  var state = this;
  var channels = state.channels;
  var isActive = state.tab === file.name;

  return h('.editor', {
    className: (isActive ? 'active' : ''),
    'ev-ace-change': aceChange(channels.fileChange),
  }, [
    aceWidget(file)
  ]);
}
