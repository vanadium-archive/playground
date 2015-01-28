var veyron = require('veyron');

veyron.init(function(err, rt) {
  if (err) throw err;

  var ctx = rt.getContext();

  rt.newClient().bindTo(ctx, 'pingpong', function(err, s) {
    if (err) throw err;

    s.ping(ctx, 'PING', function(err, pong) {
      if (err) throw err;

      console.log(pong);
      process.exit(0);
    });
  });
});
