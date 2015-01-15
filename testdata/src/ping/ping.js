var veyron = require('veyron');
var context = veyron.context;

veyron.init(function(err, rt) {
  if (err) throw err;

  var ctx = new context.Context();

  rt.bindTo(ctx, 'pingpong', function(err, s) {
    if (err) throw err;

    s.ping(ctx, 'PING', function(err, pong) {
      if (err) throw err;

      console.log(pong);
      process.exit(0);
    });
  });
});
