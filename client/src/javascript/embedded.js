module.exports = EmbeddedPlayground;

var _ = require('lodash');
var http = require('http');
var moment = require('moment');
var path = require('path');
var url = require('url');

var m = require('mercury');
var h = m.h;

var Editor = require('./editor');

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
  var state = m.state({
    activeTab: m.value(0),
    nextRunId: m.value(0),
    running: m.value(false),
    hasRun: m.value(false),
    consoleEvents: m.value([]),
    channels: {
      run: this.run.bind(this),
      reset: this.reset.bind(this),
      switchTab: this.switchTab.bind(this)
    }
  });
  m.app(el, state, this.render_.bind(this));
}

EmbeddedPlayground.prototype.renderTopBar_ = function(state) {
  var tabs = _.map(this.files_, function(file, i) {
    var selector = 'div.tab';
    if (i === state.activeTab) {
      selector += '.active';
    }
    return h(selector, {
      'ev-click': m.sendClick(state.channels.switchTab, {index: i})
    }, file.basename);
  });

  var runBtn = h('button.btn', {
    'ev-click': m.sendClick(state.channels.run)
  }, 'Run');
  var resetBtn = h('button.btn', {
    'ev-click': m.sendClick(state.channels.reset)
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
  // A single trailing newline is always ignored.
  // Ignoring the last character, check if there are any newlines in message.
  if (event.Message.slice(0, -1).indexOf('\n') !== -1) {
    // Multiline messages are marked with U+23CE and started in a new line.
    children.push('\u23ce'/* U+23CE RETURN SYMBOL */, h('br'));
  }
  children.push(h('span.message.' + (event.Stream || 'unknown'),
                  event.Message));
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

EmbeddedPlayground.prototype.switchTab = function(state, data) {
  state.activeTab.set(data.index);
};

// Sends the files to the backend, then injects the response in the console.
EmbeddedPlayground.prototype.run = function(state) {
  if (state.running()) {
    console.log('Already running', this.id_);
    return;
  }
  var runId = state.nextRunId();

  // TODO(sadovsky): Visually disable the "Run" button or change it to a "Stop"
  // button.
  state.running.set(true);
  state.hasRun.set(true);
  state.consoleEvents.set([{Message: 'Running...'}]);

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
  var that = this;

  // If the user stops the current run or resets the playground, functions
  // wrapped with ifRunActive become no-ops.
  var ifRunActive = function(cb) {
    return function() {
      if (runId === state.nextRunId()) {
        cb.apply(this, arguments);
      }
    };
  };

  var appendToConsole = function(events) {
    state.consoleEvents.set(state.consoleEvents().concat(events));
  };
  var makeEvent = function(stream, message) {
    return {Stream: stream, Message: message};
  };

  var urlp = url.parse(compileUrl);

  var options = {
    method: 'POST',
    protocol: urlp.protocol,
    hostname: urlp.hostname,
    port: urlp.port || (urlp.protocol === 'https:' ? '443' : '80'),
    path: urlp.path,
    // TODO(ivanpi): Change once deployed.
    withCredentials: false,
    headers: {
      'accept': 'application/json',
      'content-type': 'application/json'
    }
  };

  var req = http.request(options);

  var watchdog = null;
  // The heartbeat function clears the existing timeout (if any) and, if the run
  // is still active, starts a new timeout.
  var heartbeat = function() {
    if (watchdog !== null) {
      clearTimeout(watchdog);
    }
    watchdog = null;
    ifRunActive(function() {
      // TODO(ivanpi): Reduce timeout duration when server heartbeat is added.
      watchdog = setTimeout(function() {
        process.nextTick(ifRunActive(function() {
          req.destroy();
          appendToConsole(makeEvent('syserr', 'Server response timed out.'));
        }));
      }, 10500);
    })();
  };

  var endRunIfActive = ifRunActive(function() {
    that.endRun_(state);
    // Cleanup watchdog timer.
    heartbeat();
  });

  // error and close callbacks call endRunIfActive in the next tick to ensure
  // that if both events are triggered, both are executed before the run is
  // ended by either.
  req.on('error', ifRunActive(function(err) {
    console.error('Connection error: ' + err.message + '\n' + err.stack);
    appendToConsole(makeEvent('syserr', 'Error connecting to server.'));
    process.nextTick(endRunIfActive);
  }));

  // Holds partial prefix of next response line.
  var partialLine = '';

  req.on('response', ifRunActive(function(res) {
    heartbeat();
    if (res.statusCode !== 0 && res.statusCode !== 200) {
      appendToConsole(makeEvent('syserr', 'HTTP status ' + res.statusCode));
    }
    res.on('data', ifRunActive(function(chunk) {
      heartbeat();
      // Each complete line is one JSON Event.
      var eventsJson = (partialLine + chunk).split('\n');
      partialLine = eventsJson.pop();
      var events = [];
      _.forEach(eventsJson, function(line) {
        // Ignore empty lines.
        line = line.trim();
        if (line) {
          var ev;
          try {
            ev = JSON.parse(line);
          } catch (err) {
            console.error('Error parsing line: ' + line + '\n' + err.message);
            events.push(makeEvent('syserr', 'Error parsing server response.'));
            endRunIfActive();
            return false;
          }
          events.push(ev);
        }
      });
      appendToConsole(events);
    }));
  }));

  req.on('close', ifRunActive(function() {
    // Sanity check: partialLine should be empty when connection is closed.
    partialLine = partialLine.trim();
    if (partialLine) {
      console.error('Connection closed without newline after: ' + partialLine);
      appendToConsole(makeEvent('syserr', 'Error parsing server response.'));
    }
    process.nextTick(endRunIfActive);
  }));

  req.write(JSON.stringify(reqData));
  req.end();

  // Start watchdog.
  heartbeat();
};

// Clears the console and resets all editors to their original contents.
EmbeddedPlayground.prototype.reset = function(state) {
  state.consoleEvents.set([]);
  _.forEach(this.editors_, function(editor) {
    editor.reset();
  });
  this.endRun_(state);
  state.hasRun.set(false);
};

EmbeddedPlayground.prototype.endRun_ = function(state) {
  state.nextRunId.set(state.nextRunId() + 1);
  state.running.set(false);
};
