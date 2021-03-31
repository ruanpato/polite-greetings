const dotenv = require('dotenv');
const commandLineArgs = require('command-line-args');

// Setup command line options
const options = commandLineArgs([
  {
    name: 'env',
    alias: 'e',
    defaultValue: 'development',
    type: String,
  },
]);

// Setting the env file
const result2 = dotenv.config({
  path: `${__dirname}/../env/${options.env}.env`,
});

if (result2.error) {
  throw result2.error;
}
