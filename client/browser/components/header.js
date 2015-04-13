// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var h = require('mercury').h;

module.exports = {
  render: render
};

function render(state, chanels) {
  return h('header', [
    h('h1', state.title)
  ]);
}
