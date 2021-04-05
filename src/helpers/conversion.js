const textToImage = require('text2png');
const {htmlToText} = require('html-to-text');


exports.textToImageBuffer = (text, options) => {
  const imageToResponse = {};
  imageToResponse.buffer = textToImage(text, options);
  imageToResponse.header = {
    'Content-Type': 'image/png; charset=utf-8',
    'Content-Length': imageToResponse.buffer.length,
  };
  return imageToResponse;
};

exports.htmlToText = (html) => htmlToText(html, {
  wordwrap: 130,
  uppercaseHeadings: false,
});


exports.fontSizeParse = (fontSize) =>
  fontSize ? `${fontSize}px Sans` : '40px Sans';

exports.hexColorParse = (hexColor) =>
  hexColor.includes('#') ? hexColor : `#${hexColor}`;

exports.stringAsBool = (string) => {
  string = string ? string : 'false';
  if (string.toLowerCase() == 'true' || string === '1') {
    return true;
  }
  return false;
};
