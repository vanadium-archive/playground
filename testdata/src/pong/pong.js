var veyron = require('veyron');

var pingPongService = {
  ping: function($context, msg){
    console.log('['+$context.remoteBlessingStrings+'] '+msg);
    return 'PONG';
  }
};

veyron.init(function(err, rt) {
  if (err) throw err;

  rt.serve('pingpong', pingPongService, function(err) {
    if (err) throw err;
  });
});
