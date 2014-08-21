module.exports = EmbeddedPlayground;

var _ = require('lodash');
var mercury = require('mercury');

var Editor = require('./editor');

var m = mercury;
var h = mercury.h;
var request = require('superagent');

// Shows each file in a tab.
// * el: The DOM element to mount on.
// * files: List of {name, text}.
function EmbeddedPlayground(el, files) {
  this.files_ = _.map(files, function(file) {
    var type = file.name.substr(file.name.indexOf('.') + 1);
    return _.assign({}, file, {type: type});
  });
  this.editors_ = _.map(this.files_, function(file) {
    return new Editor(file.type, file.text);
  });
  this.state_ = m.struct({
    activeTab: m.value(0),
    consoleText: m.value('')
  });
  mercury.app(el, this.state_, this.render_.bind(this));
}

EmbeddedPlayground.prototype.renderConsole_ = function(text) {
  var that = this;

  var runBtn = h('button.btn', {
    'ev-click': function() {
      that.state_.consoleText.set('Running...');
      that.run();
    }
  }, 'Run');
  var shareBtn = h('button.btn', {
    'ev-click': function() {
      that.state_.consoleText.set('(Sharing is not yet implemented.)');
    }
  }, 'Share');

  var lines = _.map(text.split('\n'), function(line) {
    return h('div', line);
  });

  return h('div.console', [
    h('div.text', lines), h('div.btns', [runBtn, shareBtn])
  ]);
};

// TODO(sadovsky): It's annoying that `this.state_` and the local variable
// `state` are two different things with the same name. Need a better naming
// convention.
EmbeddedPlayground.prototype.render_ = function(state) {
  var that = this;

  var tabs = _.map(this.files_, function(file, i) {
    var selector = 'div.tab';
    if (i === state.activeTab) {
      selector += '.active';
    }
    return h(selector, {
      'ev-click': function() {
        that.state_.activeTab.set(i);
      }
    }, file.name);
  });
  var editors = _.map(this.editors_, function(editor, i) {
    var properties = {};
    if (i !== state.activeTab) {
      // Use "visibility: hidden" rather than "display: none" because the latter
      // causes the editor to initialize lazily and thus flicker when it's first
      // opened.
      properties['style'] = {visibility: 'hidden'};
    }
    return h('div.editor', properties, editor);
  });
  // TODO(sadovsky): Make the console a proper component with its own render
  // method?
  var consoleEl = this.renderConsole_(state.consoleText);
  return h('div.pg', [
    h('div', tabs), h('div.editors', editors), consoleEl
  ]);
};

// Sends the files to the backend, then injects the response in the console.
EmbeddedPlayground.prototype.run = function() {
  var that = this;

  var compileUrl = 'http://playground.envyor.com:8181/compile';

  // Uncomment the following line for testing. Instructions for how to run the
  // compile server locally are in go/src/veyron/tools/playground/README.md.
  //compileUrl = 'http://localhost:8181/compile';

  var req = {
    files: _.map(this.files_, function(file, i) {
      var editor = that.editors_[i];
      return {
        Name: file.name,
        Body: editor.getText()
      };
    }),
    Identities: []
  };

  var state = this.state_;
  request
      .post(compileUrl)
      .type('json')
      .accept('json')
      .send(req)
      .end(function(err, res) {
        if (err) {
          return console.error(err);
        }
        if (res.error) {
          return console.error(res.error);
        }
        if (res.body.Errors) {
          return state.consoleText.set(res.body.Errors);
        }
        if (res.body.Events && res.body.Events[0]) {
          // Currently only sends one event.
          return state.consoleText.set(res.body.Events[0].Message);
        }
      });
};
