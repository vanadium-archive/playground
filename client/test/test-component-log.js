// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var test = require('tape');
var log = require('../browser/components/log');

test('log(data)', function(t) {
  var data = {
    File: '',
    Message: 'Response finished↵',
    Stream: 'debug',
    Timestamp: 1428710996621822700
  };
  var state = log(data);
  var value = state();

  t.deepEqual(value, {
    file: '',
    message: data.Message,
    stream: data.Stream,
    timestamp: data.Timestamp / 1e6
  });
  t.end();
});
