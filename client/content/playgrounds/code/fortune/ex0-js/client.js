var veyron = require('veyron');
var context = veyron.context;

/**
 * Create a Vanadium runtime using the configuration defined in config.js,
 * and bind it to the bakery/cookie/fortune service.
 */
veyron.init(function(err, rt){
  if (err) { return error(err); }

  var ctx = new context.Context();

  retryBindTo(ctx, rt, function(err, fortuneService) {
    if (err) { return error(err); }

    fortuneService.getRandomFortune(ctx, function(err, fortune) {
      if (err) { return error(err); }

      console.log(fortune);
      process.exit(0);
    });
  });
});

function retryBindTo(ctx, rt, cb) {
  rt.bindTo(ctx, 'bakery/cookie/fortune', function(err, fortuneService) {
    if (err) {
      // Try again in 100ms
      return setTimeout(function(){
        retryBindTo(ctx, rt, cb);
      }, 100);
    }

    cb(null, fortuneService);
  });
}

function error(err) {
  console.error(err);
  process.exit(1);
}
