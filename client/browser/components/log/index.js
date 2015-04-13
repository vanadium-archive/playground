// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var hg = require('mercury');
var normalize = require('./normalize');

module.exports = log;
module.exports.render = require('./render');

function log(data) {
  data = normalize(data);

  var state = hg.struct({
    message: data.message,
    file: data.file,
    stream: data.stream,
    timestamp: data.timestamp
  });

  return state;
}
