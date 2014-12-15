var veyron = require('veyron');
var context = veyron.context;

veyron.init(function(err, rt){
  if (err) throw err;

  rt.bindTo('pingpong', function(err, s){
    if (err) throw err;

    var ctx = new context.Context();

    s.ping(ctx, 'PING', function(err, pong){
      if (err) throw err;

      console.log(pong);
      process.exit(0);
    });
  });
});
