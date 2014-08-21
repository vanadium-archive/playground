module.exports = Editor;

// NOTE(sadovsky): We considered both Ace and CodeMirror, but CodeMirror appears
// to be incompatible with Bootstrap and has other weird issues, e.g. its
// injected editor divs can obscure other DOM elements.

var ace = require('brace');
require('brace/mode/javascript');
require('brace/mode/golang');
require('brace/theme/monokai');

// Mercury widget that wraps a code editor.
// * type: Type of code. Currently either 'js' or 'go'.
// * text: Initial code.
function Editor(type, text) {
  this.type_ = type;
  this.text_ = text;
  this.aceEditor_ = null;
}

// This tells Mercury to treat Editor as a widget and not try to render its
// internals.
Editor.prototype.type = 'Widget';

Editor.prototype.init = function() {
  console.log('EditorWidget.init');
  var el = document.createElement('div');
  this.mount(el, this.type_, this.text_);
  return el;
};

Editor.prototype.update = function() {
  console.log('EditorWidget.update');
};

Editor.prototype.getText = function() {
  return this.aceEditor_.getSession().getValue();
};

// Creates a new Ace editor instance and mounts it on a DOM node.
// * el: The DOM node to mount on.
// * type: Type of code. Currently either 'js' or 'go'.
// * text: Initial code.
Editor.prototype.mount = function(el, type, text) {
  var editor = this.aceEditor_ = ace.edit(el);
  editor.setTheme('ace/theme/monokai');
  editor.setFontSize(15);

  var session = editor.getSession();
  switch (type) {
  case 'go':
    session.setMode('ace/mode/golang');
    break;
  case 'js':
    session.setMode('ace/mode/javascript');
    break;
  default:
    throw new Error('Language type not supported: ' + type);
  }

  session.setValue(text);

  // Disable syntax checking. The UI is annoying and only works for JS anyways.
  session.setOption('useWorker', false);
};
