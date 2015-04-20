// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
var h = require('mercury').h;
var moment = require('moment');

module.exports = render;

// This is expected to be called an iterator fn passed to logs.map(render)
function render(state, index, logs) {
  var time = moment(state.timestamp).format('MMM D HH:mm:ss SSS');
  var stream = state.stream || 'unknown';

  return h('.log', {
    className: stream
  }, [
    h('.meta', [
      h('.source', [
        h('span', state.file || 'system: '),
        h('span.stream', (! state.file) ? stream : '')
      ]),
      h('.time', '[' + time + ']')
    ]),
    h('.message', [
      h('pre', [
        h('code', state.message)
      ])
    ])
  ]);
}
