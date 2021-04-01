const yamljs = require('yamljs');
const path = require('path');

const getSwaggerFilePath = () =>
  path.join(process.cwd(), '/docs/apiDocs.yml');

const getYamlAsJson = (filePath) => yamljs.load(filePath);

exports.getSwaggerDocument = () => {
  try {
    return getYamlAsJson(getSwaggerFilePath());
  } catch (err) {
    return 'Erro';
  }
};
