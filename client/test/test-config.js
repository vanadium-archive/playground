// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var test = require('tape');
var config = require('../browser/config');

test('config(key, value)', function(t) {
  var url = 'https://playground.staging.v.io/api';

  config('api-url', url);
  var value = config('api-url');

  t.equal(value, url);
  t.end();
});
