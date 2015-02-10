module.exports = Spinner;

var spinjs = require('spin.js');

// Mercury widget for displaying a spinner over an element.
function Spinner() {
  this.spinner_ = new spinjs({
    className: 'spinner-internal',
    'z-index': 2000000000,
    color: '#ffffff'
  });
}

// This tells Mercury to treat Spinner as a widget and not try to render its
// internals.
Spinner.prototype.type = 'Widget';

Spinner.prototype.init = function() {
  console.log('SpinnerWidget.init');
  var el = document.createElement('div');
  el.setAttribute('class', 'spinner-overlay');
  this.spinner_.spin(el);
  return el;
};

Spinner.prototype.update = function(prev, el) {
  console.log('SpinnerWidget.update');
};
