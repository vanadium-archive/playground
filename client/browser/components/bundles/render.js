// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var debug = require('debug')('components:bundles');
var h = require('mercury').h;
var anchor = require('../../router/anchor');
var toArray = require('../../util').toArray;

module.exports = render;

function render(state) {
  debug('update %o', state);

  var bundles = toArray(state);

  if (bundles.length === 0) {
    return h('p', 'Loading...');
  } else {
    return h('ul.bundles', bundles.map(li));
  }
}

function li(bundle) {
  return h('li', [
    anchor({ href: '/' + bundle.uuid }, bundle.uuid)
  ]);
}
