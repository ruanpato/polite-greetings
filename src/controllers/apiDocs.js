const fs = require('fs');
const path = require('path');
const yamljs = require('js-yaml');

const docsLocation = path.join(process.cwd(), '/docs');

exports.convertYamltoJson = () => {
  const swaggerYamlFile = (`${docsLocation}/politeGreetings.yml`.replace('/src/controllers', ''));
  const swaggerJsFile = (`${docsLocation}/politeGreetings.json`.replace('/src/controllers', ''));
  fs.writeFileSync(swaggerJsFile, JSON.stringify(yamljs.load(fs.readFileSync(swaggerYamlFile, { encoding: 'utf-8' })), null, 2));
};
