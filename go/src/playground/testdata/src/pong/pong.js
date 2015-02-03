var veyron = require('veyron');
var vom = require('vom');

var pingPongService = {
  ping: function(ctx, msg) {
    console.log('[' + ctx.remoteBlessingStrings + '] ' + msg);
    return 'PONG';
  },
  // TODO(alexfandrianto): The correct way to do this is to generate the JS code
  // from the VDL file and inherit from the generated service stub.
  _serviceDescription: {
    methods: [
      {
        name: 'Ping',
        inArgs: [
          {
            name: 'msg',
            type: vom.Types.STRING
          }
        ],
        outArgs: [
          {
            type: vom.Types.STRING
          }
        ]
      }
    ]
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
