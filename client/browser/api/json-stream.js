// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var through2 = require('through2');

module.exports = create;

function create() {
  return through2.obj(write);
}

function write(buffer, enc, callback) {
  if (buffer.length === 0) {
    return callback();
  }

  var json;
  var err;

  try {
    json = JSON.parse(buffer);
  } catch (err) {
    err.data = buffer;
  }

  callback(err, json);
}
