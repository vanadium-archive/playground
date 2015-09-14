// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var vanadium = require('vanadium');

vanadium.init(function(err, rt) {
  if (err) throw err;

  var ctx = rt.getContext();

  console.log('Binding to service');
  rt.getClient().bindTo(ctx, 'pingpong', function(err, s) {
    if (err) throw err;

    console.log('Pinging');
    s.ping(ctx, 'PING', function(err, pong) {
      if (err) throw err;

      console.log(pong);
      process.exit(0);
    });
  });
});
