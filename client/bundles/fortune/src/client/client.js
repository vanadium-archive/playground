// index=2
var vanadium = require('vanadium');

/**
 * Create a Vanadium runtime using the configuration defined in config.js,
 * and bind it to the bakery/cookie/fortune service.
 */
vanadium.init(function(err, rt) {
  if (err) { return error(err); }

  var ctx = rt.getContext();
  var client = rt.newClient();

  console.info('Binding to service');
  retryBindTo(ctx, client, function(err, fortuneService) {
    if (err) { return error(err); }

    console.info('Issuing request');
    fortuneService.getRandomFortune(ctx, function(err, fortune) {
      if (err) { return error(err); }

      console.log('Received: ' + fortune);
      process.exit(0);
    });
  });
});

function retryBindTo(ctx, client, cb) {
  client.bindTo(ctx, 'bakery/cookie/fortune', function(err, fortuneService) {
    if (err) {
      console.error(err + '\nRetrying in 100ms...');
      return setTimeout(function() {
        retryBindTo(ctx, client, cb);
      }, 100);
    }

    cb(null, fortuneService);
  });
}

function error(err) {
  console.error(err);
  process.exit(1);
}
