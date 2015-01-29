var veyron = require('veyron');

var pingPongService = {
  ping: function(ctx, msg) {
    console.log('[' + ctx.remoteBlessingStrings + '] ' + msg);
    return 'PONG';
  }
};

veyron.init(function(err, rt) {
  if (err) throw err;

  console.log('Starting server');
  rt.newServer().serve('pingpong', pingPongService, function(err) {
    if (err) throw err;

    console.log('Serving pingpong');
  });
});
