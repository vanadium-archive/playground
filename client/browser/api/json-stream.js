// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Using jxson/split#ignore-trailing until it is merged upstream.
//
// SEE: https://github.com/dominictarr/split/pull/15
var split = require('split');

module.exports = JSONStream;

function JSONStream() {
  return split(parse, null, { trailing: false });
}

function parse(buffer) {
  if (buffer.length === 0) {
    return undefined;
  } else {
    return JSON.parse(buffer);
  }
}
