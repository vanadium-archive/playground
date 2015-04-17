// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var inherits = require('util').inherits;
var vanadium = require('vanadium');

var pingpong = require('../pingpong');

function PingPongService() {}

inherits(PingPongService, pingpong.PingPong);

PingPongService.prototype.ping = function(ctx, serverCall, message) {
  var secCall = serverCall.securityCall;
  console.log('[' + secCall.remoteBlessingStrings + '] ' + message);
  return Promise.resolve('PONG');
};

var pingPongService = new PingPongService();

vanadium.init(function(err, rt) {
  if (err) throw err;

  console.log('Starting server');
  rt.newServer().serve('pingpong', pingPongService, function(err) {
    if (err) throw err;

    console.log('Serving pingpong');
  });
});
