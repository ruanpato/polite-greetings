const axios = require('axios').default;

exports.get = async (url, responseType) => axios({
  method: 'get',
  url,
  responseType,
});
