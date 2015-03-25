var _ = require('lodash');
var fs = require('fs');
var glob = require('glob');
var path = require('path');

module.exports = {run: run};

function usage() {
  console.error('Usage: pgbundle [options] <glob_file> <root_path>');
  console.error('Arguments: <glob_file>: Path to file containing a list of' +
    ' glob patterns, one per line. The bundle includes only files with path' +
    ' suffixes matching one of the globs. Each glob must match at least one' +
    ' file, otherwise bundling fails with a non-zero exit code.');
  console.error('           <root_path>: Path to directory where files' +
    ' matching glob patterns are taken from.');
  console.error('Options: --verbose: Enable verbose output.');
  console.error('         --empty: Omit file contents in bundle, include only' +
    ' paths and metadata.');
  process.exit(1);
}

// Strip the first encountered "// +build ignore" line in the file and return
// the remaining lines.
function stripBuildIgnore(lines) {
  var found = false;
  return _.filter(lines, function(line) {
    if (!found && (_.trim(line) === '// +build ignore')) {
      found = true;
      return false;
    }
    return true;
  });
}

// Strip the first encountered "// pg-index=<num>" line in the file and return
// the index value and remaining lines.
function getIndex(lines) {
  var index = null;
  lines = _.filter(lines, function(line) {
    var match = _.trim(line).match(/^\/\/\s*pg-index=(\d+)/);
    if (!index && match && match[1]) {
      index = match[1];
      return false;
    }
    return true;
  });
  return {
    index: index ? Number(index) : Infinity,
    lines: lines
  };
}

// Strip all blank lines at the beginning of the file.
function stripLeadingBlankLines(lines) {
  var nb = 0;
  for (; nb < lines.length && _.trim(lines[nb]) === ''; ++nb) /* no-op */;
  return _.slice(lines, nb);
}

// Main function.
function run() {
  // Get the flags and positional arguments from process.argv.
  var argv = require('minimist')(process.argv.slice(2), {
    boolean: ['verbose', 'empty']
  });

  // Make sure the glob file and the root path path are specified.
  if (!argv._ || argv._.length !== 2) {
    return usage();
  }

  var globFile = argv._[0];
  var dir = argv._[1];
  // Read glob file, filtering out empty lines.
  var patterns = _.filter(
    fs.readFileSync(globFile, {encoding: 'utf8'}).split('\n'));
  // The root path must be a directory.
  if (!fs.lstatSync(dir).isDirectory()) {
    return usage();
  }

  var unmatched = [];

  // Apply each glob pattern to the directory.
  var relpaths = _.flatten(_.map(patterns, function(pattern) {
    var match = glob.sync('**/' + pattern, {
      cwd: dir,
      nodir: true
    });
    if (match.length === 0) {
      unmatched.push(pattern);
    }
    return match;
  }));

  // If any pattern matched zero files, halt with a non-zero exit code.
  // TODO(ivanpi): Allow optional patterns, e.g. prefixed by '?'?
  if (unmatched.length > 0) {
    console.warn('Error bundling "%s": unmatched patterns %j', dir, unmatched);
    process.exit(2);
  }

  var out = {files: []};

  // Loop over each file.
  _.each(relpaths, function(relpath) {
    var abspath = path.resolve(dir, relpath);
    var lines = fs.readFileSync(abspath, {encoding: 'utf8'}).split('\n');

    lines = stripBuildIgnore(lines);
    var indexAndLines = getIndex(lines);
    var index = indexAndLines.index;
    lines = indexAndLines.lines;
    lines = stripLeadingBlankLines(lines);

    out.files.push({
      name: relpath,
      body: argv.empty ? '' : lines.join('\n'),
      index: index
    });
  });

  out.files = _.sortBy(out.files, 'index');

  // Drop the index fields -- we don't need them anymore.
  out.files = _.map(out.files, function(f) {
    return _.omit(f, 'index');
  });

  // Write the bundle to stdout.
  process.stdout.write(JSON.stringify(out) + '\n');

  if (argv.verbose) {
    console.warn('Bundled "%s" using "%s"', dir, globFile);
  }
}
