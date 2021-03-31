const dotenv = require('dotenv');
const path = require('path');
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

const envFilePath = path.join(`${process.cwd()}/env/${options.env}.env`);
// Setting the env file
const result2 = dotenv.config({
  path: envFilePath,
});

if (result2.error) {
  throw result2.error;
}
