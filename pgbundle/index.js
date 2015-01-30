var _ = require('lodash');
var fs = require('fs');
var glob = require('glob');
var path = require('path');

// Filename to write the data to.
var BUNDLE_NAME = 'bundle.json';

module.exports = {run: run};

// TODO(nlacasse): Improve this.
function usage() {
  console.log('Usage: pgbundle [options] <path> [<path> <path> ...]');
  console.log('Options: --verbose: Enable verbose output.');
  process.exit(1);
}

// If the first line is "// +build ignore", strip the line and return the
// remaining lines.
function stripBuildIgnore(lines) {
  if (lines.length > 0 && _.trim(lines[0]) === '// +build ignore') {
    return _.rest(lines);
  }
  return lines;
}

// Strip all blank lines at the beginning of the file.
function stripLeadingBlankLines(lines) {
  var nb = 0;
  for (; nb < lines.length && _.trim(lines[nb]) === ''; ++nb) /* no-op */;
  return _.slice(lines, nb);
}

// If the first line is an index comment, strip the line and return the index
// and remaining lines.
function getIndex(lines) {
  var index = null;
  var match = lines.length > 0 &&
              _.trim(lines[0]).match(/^\/\/\s*index=(\d+)/);
  if (match && match[1]) {
    index = match[1];
    lines = _.rest(lines);
  }
  return {
    index: index,
    lines: lines
  };
}

function shouldIgnore(fileName) {
  // Ignore directories.
  if (_.last(fileName) === '/') {
    return true;
  }
  // Ignore bundle files.
  if (fileName === BUNDLE_NAME) {
    return true;
  }
  // Ignore generated .vdl.go and .vdl.js files.
  if ((/\.vdl\.(go|js)$/i).test(fileName)) {
    return true;
  }
  // Ignore files inside "bin" and "pkg" directories.
  if (fileName.indexOf('bin/') === 0 ||
      fileName.indexOf('pkg/') === 0) {
    return true;
  }
  return false;
}

// Main function.
function run() {
  // Get the paths from process.argv.
  var argv = require('minimist')(process.argv.slice(2));
  var dirs = argv._;

  // Make sure there is at least one path.
  if (!dirs || dirs.length === 0) {
    return usage();
  }

  // Loop over each path.
  _.each(dirs, function(dir) {
    var relpaths = glob.sync('**', {
      cwd: dir,
      mark: true  // Add a '/' char to directory matches.
    });

    if (relpaths.length === 0) {
      return usage();
    }

    var out = {files: []};

    // Loop over each file.
    _.each(relpaths, function(relpath) {
      if (shouldIgnore(relpath)) {
        return;
      }

      var abspath = path.resolve(dir, relpath);
      var text = fs.readFileSync(abspath, {encoding: 'utf8'});

      var lines = text.split('\n');
      lines = stripBuildIgnore(lines);
      lines = stripLeadingBlankLines(lines);
      var indexAndLines = getIndex(lines);
      var index = indexAndLines.index;
      lines = indexAndLines.lines;

      out.files.push({
        name: relpath,
        body: lines.join('\n'),
        index: index
      });
    });

    out.files = _.sortBy(out.files, 'index');

    // Drop the index fields -- we don't need them anymore.
    out.files = _.map(out.files, function(f) {
      return _.omit(f, 'index');
    });

    // Write the bundle.json.
    var outFile = path.resolve(dir, BUNDLE_NAME);
    fs.writeFileSync(outFile, JSON.stringify(out));

    if (argv.verbose) {
      console.log('Wrote ' + outFile);
    }
  });
}
