const textToImage = require('text2png');

exports.getImageToReturnResponse = (text, options) => {
  const imageToResponse = {};
  imageToResponse.buffer = textToImage(text, options);
  imageToResponse.header = {
    'Content-Type': 'image/png; charset=utf-8',
    'Content-Length': imageToResponse.buffer.length,
  };
  return imageToResponse;
};
