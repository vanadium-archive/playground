// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var hg = require('mercury');

module.exports = bundles;
module.exports.render = require('./render');

function bundles() {
  return hg.varhash({});
}
