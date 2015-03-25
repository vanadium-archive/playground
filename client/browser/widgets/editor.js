// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
  // Every explicitly created (not copied) editor has a unique nonce.
  this.nonce_ = Editor.nonceSeed_;
  Editor.nonceSeed_ = (Editor.nonceSeed_ + 1) & 0x7fffffff;
  this.aceEditor_ = null;
}

Editor.nonceSeed_ = 0;

// This tells Mercury to treat Editor as a widget and not try to render its
// internals.
Editor.prototype.type = 'Widget';

Editor.prototype.init = function() {
  console.log('EditorWidget.init');
  var el = document.createElement('div');
  this.mount(el, this.type_, this.text_);
  return el;
};

Editor.prototype.update = function(prev, el) {
  console.log('EditorWidget.update');
  // If update is called with the currently mounted Editor instance or its
  // copy (detected by nonce), remount would reset any edits. Remount should
  // only happen if a new editor instance (with a new nonce) is explicitly
  // created.
  if (this.nonce_ !== prev.nonce_) {
    this.mount(el, this.type_, this.text_);
  }
};

Editor.prototype.getText = function() {
  return this.aceEditor_.getValue();
};

Editor.prototype.reset = function() {
  // The '-1' argument puts the cursor at the document start.
  this.aceEditor_.setValue(this.text_, -1);
};

// Creates a new Ace editor instance and mounts it on a DOM node.
// * el: The DOM node to mount on.
// * type: Type of code. Currently either 'js' or 'go'.
// * text: Initial code.
Editor.prototype.mount = function(el, type, text) {
  var editor = this.aceEditor_ = ace.edit(el);
  editor.setTheme('ace/theme/monokai');

  var session = editor.getSession();
  switch (type) {
  case 'go':
    session.setMode('ace/mode/golang');
    break;
  case 'vdl':
    session.setMode('ace/mode/golang');
    break;
  case 'json':
    session.setMode('ace/mode/javascript');
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
