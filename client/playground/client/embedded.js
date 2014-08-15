module.exports = EmbeddedPlayground;

var _ = require('lodash');
var mercury = require('mercury');

var editor = require('./editor');

var m = mercury;
var h = mercury.h;

// Mercury widget that wraps a code editor.
// * type: Type of code. Currently either 'js' or 'go'.
// * text: Initial code.
function EditorWidget(type, text) {
  this.type_ = type;
  this.text_ = text;
}

EditorWidget.prototype = {
  type: 'Widget',
  init: function() {
    console.log('EditorWidget.init');
    var el = document.createElement('div');
    editor.mount(el, this.type_, this.text_);
    return el;
  },
  update: function() {
    console.log('EditorWidget.update');
  }
};

// Shows each file in a tab.
// * el: The DOM element to mount on.
// * files: List of {name, text}.
function EmbeddedPlayground(el, files) {
  this.files_ = _.map(files, function(file) {
    var type = file.name.substr(file.name.indexOf('.') + 1);
    return _.assign({}, file, {type: type});
  });
  this.state_ = m.struct({
    activeTab: m.value(0),
    consoleText: m.value('')
  });
  mercury.app(el, this.state_, this.render_.bind(this));
}

EmbeddedPlayground.prototype = {
  // TODO(sadovsky): It's annoying that `this.state_` and the local variable
  // `state` are two different things with the same name. Need a better naming
  // convention.
  render_: function(state) {
    var that = this;
    var files = this.files_;

    var tabs = _.map(files, function(file, i) {
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
    var editors = _.map(files, function(file, i) {
      var properties = {};
      if (i !== state.activeTab) {
        // Use "visibility: hidden" rather than "display: none" because the
        // latter causes the editor to initialize lazily and thus flicker when
        // it's first opened.
        properties['style'] = {visibility: 'hidden'};
      }
      return h('div.editor', properties,
               new EditorWidget(file.type, file.text));
    });
    var runBtn = h('button.btn', {
      'ev-click': function() {
        that.state_.consoleText.set('Run');
      }
    }, 'Run');
    var shareBtn = h('button.btn', {
      'ev-click': function() {
        that.state_.consoleText.set('Share');
      }
    }, 'Share');
    var consoleEl = h('div.console.clearfix', [
      h('span', state.consoleText), shareBtn, runBtn
    ]);
    return h('div.pg', [
      h('div', tabs), h('div.editors', editors), consoleEl
    ]);
  }
};
