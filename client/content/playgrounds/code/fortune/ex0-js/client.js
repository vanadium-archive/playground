var veyron = require('veyron');
var context = veyron.context;

/**
 * Create a Veyron runtime using the configuration defined in config.js,
 * and bind it to the bakery/cookie/fortune service.
 */
veyron.init({}, function(err, rt){
  if (err) { return error(err); }

  retryBindTo(rt, function(err, fortuneService) {
    if (err) { return error(err); }

    var ctx = new context.Context();

    fortuneService.getRandomFortune(ctx, function(err, fortune) {
      if (err) { return error(err); }

      console.log(fortune);
      process.exit(0);
    });
  });
});

function retryBindTo(rt, cb) {
  rt.bindTo('bakery/cookie/fortune', function(err, fortuneService) {
    if (err) {
      // Try again in 100ms
      return setTimeout(function(){
        retryBindTo(rt, cb);
      }, 100);
    }

    cb(null, fortuneService);
  });
}

function error(err) {
  console.error(err);
  process.exit(1);
}
