// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var h = require('mercury').h;
var moment = require('moment');

module.exports = render;

function render(state) {
  var date = moment(state.timestamp).format('H:mm:ss.SSS');

  var children = [
    h('span.timestamp', date + ' '),
    h('span.filename', state.file ? state.file + ': ' : '')
  ];

  // TODO(jasoncampbell): render in a pre tag instead

  // A single trailing newline is always ignored.
  // Ignoring the last character, check if there are any newlines in message.
  if (state.message.slice(0, -1).indexOf('\n') !== -1) {
    children.push('\u23ce'); // U+23CE RETURN SYMBOL
    children.push('br');
  }

  var message = h('span.message', {
    className: state.stream || 'unknown'
  }, state.message);

  children.push(message);

  return h('.log', children);
}
