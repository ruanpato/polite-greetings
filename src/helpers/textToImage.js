const textToImage = require('text2png');

exports.getImageToReturnResponse = (text, options) => {
  const imageToResponse = {};
  imageToResponse.buffer = textToImage(text, options);
  imageToResponse.header = {
    'Content-Type': 'image/png',
    'Content-Length': imageToResponse.buffer.length,
  };
  return imageToResponse;
};
