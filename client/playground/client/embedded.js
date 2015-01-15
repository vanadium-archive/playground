module.exports = EmbeddedPlayground;

var _ = require('lodash');
var http = require('http');
var mercury = require('mercury');
var moment = require('moment');
var path = require('path');
var url = require('url');

var Editor = require('./editor');

var m = mercury;
var h = mercury.h;

// Shows each file in a tab.
// * el: The DOM element to mount on.
// * id: Identifier for this playground instance, used in debug messages.
// * files: List of {name, body}, as written by bundler.
function EmbeddedPlayground(el, id, files) {
  this.id_ = id;
  this.files_ = _.map(files, function(file) {
    return _.assign({}, file, {
      basename: path.basename(file.name),
      type: path.extname(file.name).substr(1)
    });
  });
  this.editors_ = _.map(this.files_, function(file) {
    return new Editor(file.type, file.body);
  });
  this.state_ = m.struct({
    activeTab: m.value(0),
    nextRunId: m.value(0),
    running: m.value(false),
    hasRun: m.value(false),
    consoleEvents: m.value([])
  });
  mercury.app(el, this.state_, this.render_.bind(this));
}

EmbeddedPlayground.prototype.renderTopBar_ = function(state) {
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
    }, file.basename);
  });

  var runBtn = h('button.btn', {
    'ev-click': that.run.bind(that)
  }, 'Run');
  var resetBtn = h('button.btn', {
    'ev-click': that.reset.bind(that)
  }, 'Reset');

  return h('div.top-bar', [h('div', tabs), h('div.btns', [runBtn, resetBtn])]);
};

EmbeddedPlayground.prototype.renderEditors_ = function(state) {
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

  return h('div.editors', editors);
};

function renderConsoleEvent(event) {
  var children = [];
  if (event.Timestamp) {
    // Convert UTC to local time.
    var t = moment(event.Timestamp / 1e6);
    children.push(h('span.timestamp', t.format('H:mm:ss.SSS') + ' '));
  }
  if (event.File) {
    children.push(h('span.filename', path.basename(event.File) + ': '));
  }
  children.push(h('span.' + (event.Stream || 'unknown'), event.Message));
  return h('div', children);
}

EmbeddedPlayground.prototype.renderConsole_ = function(state) {
  if (state.hasRun) {
    return h('div.console.open', [
      h('div.text', _.map(state.consoleEvents, renderConsoleEvent))
    ]);
  }
  return h('div.console');
};

EmbeddedPlayground.prototype.render_ = function(state) {
  return h('div.pg', [
    this.renderTopBar_(state),
    this.renderEditors_(state),
    this.renderConsole_(state)
  ]);
};

// Sends the files to the backend, then injects the response in the console.
EmbeddedPlayground.prototype.run = function() {
  if (this.state_.running()) {
    console.log('Already running', this.id_);
    return;
  }
  var runId = this.state_.nextRunId();

  // TODO(sadovsky): Visually disable the "Run" button or change it to a "Stop"
  // button.
  this.state_.running.set(true);
  this.state_.hasRun.set(true);
  this.state_.consoleEvents.set([{Message: 'Running...'}]);

  var myUrl = url.parse(window.location.href, true);
  var pgaddr = myUrl.query.pgaddr;
  if (pgaddr) {
    console.log('Using pgaddr', pgaddr);
  } else {
    pgaddr = 'https://staging.v.io/playground';
  }
  var compileUrl = pgaddr + '/compile';
  if (myUrl.query.debug === '1') {
    compileUrl += '?debug=1';
  }

  var editors = this.editors_;
  var reqData = {
    files: _.map(this.files_, function(file, i) {
      var editor = editors[i];
      return {
        Name: file.name,
        Body: editor.getText()
      };
    }),
    Identities: []
  };

  // TODO(sadovsky): To deal with cached responses, shift timestamps (based on
  // current time) and introduce a fake delay. Also, switch to streaming
  // messages, for usability.
  var that = this, state = this.state_;

  // If the user stops the current run or resets the playground, functions
  // wrapped with ifRunActive become no-ops.
  var ifRunActive = function(foo) {
    return function() {
      if (runId === state.nextRunId()) {
        foo.apply(this, arguments);
      }
    };
  };

  var appendToConsole = function(events) {
    state.consoleEvents.set(state.consoleEvents().concat(events));
  };
  var makeEvent = function(stream, message) {
    return {Stream: stream, Message: message};
  };
  var endRunIfActive = ifRunActive(function() {
    that.endRun_();
  });

  var optp = url.parse(compileUrl);

  var options = {
    method: 'POST',
    protocol: optp.protocol,
    hostname: optp.hostname,
    port: optp.port || (optp.protocol === 'https:' ? '443' : '80'),
    path: optp.path,
    // TODO(ivanpi): Change once deployed.
    withCredentials: false,
    headers: {
      'accept': 'application/json',
      'content-type': 'application/json'
    }
  };

  var req = http.request(options);

  // error and close callbacks call endRunIfActive in the next tick to ensure
  // that if both events are triggered, both are executed before the run is
  // ended by one of them.
  req.on('error', ifRunActive(function(err) {
    console.log(err);
    appendToConsole(makeEvent('syserr', 'Error connecting to server.'));
    process.nextTick(endRunIfActive);
  }));

  req.on('close', ifRunActive(function() {
    process.nextTick(endRunIfActive);
  }));

  req.on('response', ifRunActive(function(res) {
    if (res.statusCode !== 0 && res.statusCode !== 200) {
      appendToConsole(makeEvent('syserr', 'HTTP status ' + res.statusCode));
    }
    // Holds partial prefix of next line.
    var line = {buffer: ''};
    res.on('data', ifRunActive(function(chunk) {
      // Each complete line is one JSON Event.
      var eventsJson = (line.buffer + chunk).split('\n');
      line.buffer = eventsJson.pop();
      var events = [];
      _.forEach(eventsJson, function(el) {
        // Ignore empty and invalid lines.
        if (el && el.charAt(0) === '{') {
          try {
            events.push(JSON.parse(el));
          } catch (err) {
            console.error(err);
            events.push(makeEvent('syserr', 'Error parsing server response.'));
            endRunIfActive();
            return false;
          }
        }
      });
      appendToConsole(events);
    }));
  }));

  req.write(JSON.stringify(reqData));
  req.end();
};

// Clears the console and resets all editors to their original contents.
EmbeddedPlayground.prototype.reset = function() {
  this.state_.consoleEvents.set([]);
  _.forEach(this.editors_, function(editor) {
    editor.reset();
  });
  this.endRun_();
  this.state_.hasRun.set(false);
};

EmbeddedPlayground.prototype.endRun_ = function() {
  this.state_.nextRunId.set(this.state_.nextRunId() + 1);
  this.state_.running.set(false);
};
