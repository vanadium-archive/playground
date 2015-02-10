var _ = require('lodash');
var path = require('path');
var superagent = require('superagent');

var Playground = require('./playground');

_.forEach(document.querySelectorAll('.playground'), function(el) {
  var srcdir = el.getAttribute('data-srcdir');
  console.log('Creating playground', srcdir);

  fetchBundle(srcdir, function(err, bundle) {
    if (err) {
      el.innerHTML = '<div class="error"><p>Playground error.' +
        '<br>Bundle not found: <strong>' + srcdir + '</strong></p></div>';
      return;
    }
    new Playground(el, srcdir, bundle);  // jshint ignore:line
  });
});

function fetchBundle(loc, cb) {
  var basePath = '/bundles';
  console.log('Fetching bundle', loc);
  superagent
    .get(path.join(basePath, loc, 'bundle.json'))
    .accept('json')
    .end(function(err, res) {
      if (err) {
        return cb(err);
      }
      if (res.error) {
        return cb(res.error);
      }
      cb(null, res.body);
    });
}
