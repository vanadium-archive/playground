// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var hg = require('mercury');
var h = require('mercury').h;

module.exports = toast;
module.exports.render = render;

function toast(options) {
  options = options || {};

  var state = hg.state({
    message: hg.value(options.message || ''),
    active: hg.value(options.message ? true : false),
  });

  return state;
}

function render(state) {
  var attributes = {
    className: state.active ? 'active' : ''
  };

  var children = [
    h('p', state.message)
  ];

  return h('.toast', attributes, children);
}
