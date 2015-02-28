var vanadium = require('vanadium');

vanadium.init(function(err, rt) {
  if (err) throw err;

  var ctx = rt.getContext();

  console.log('Binding to service');
  rt.newClient().bindTo(ctx, 'pingpong', function(err, s) {
    if (err) throw err;

    console.log('Pinging');
    s.ping(ctx, 'PING', function(err, pong) {
      if (err) throw err;

      console.log(pong);
      process.exit(0);
    });
  });
});
