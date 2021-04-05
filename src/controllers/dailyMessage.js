const {Validator} = require('node-input-validator');

const log = require('../helpers/lib/log/log');
const convert = require('../helpers/conversion');
const request = require('../helpers/request');

exports.getDailyMessage = async (req, res, next) => {
  const dailyObject = req.query;

  const validateParams = new Validator(dailyObject, {
    fontColor: 'hexColor',
    backgroundColor: 'hexColor',
  });

  const paramsIsValid = await validateParams.check();

  if (!paramsIsValid) {
    return res.status(422).json(validateParams.errors);
  }

  try {
    const image = await hackerLawsDaily(dailyObject, res);
    res.set(image.header);
    return res.status(200).end(image.buffer);
  } catch (err) {
    log.error('Error:  ', err);
    return res.status(500).json({
      mesage: 'Internal Server Error',
    });
  }
};

const hackerLawsDaily = async (dailyObject, res) => {
  res.set({
    'Cache-control': `public, max-age=${2000}`,
  });
  const webApiHackerLawDaily = 'https://tips.darekkay.com/json/hacker-laws-en.json';

  const hackerDaily = (await request.get(webApiHackerLawDaily, 'json')).data;

  let dailyMessage = convert.htmlToText(hackerDaily.title);
  dailyMessage += `\n${'-'.repeat(160)}\n${
    convert.htmlToText(hackerDaily.content)}`;
  const imageOptions = {
    font: convert.fontSizeParse(dailyObject.fontSize),
    backgroundColor: convert.hexColorParse(
        dailyObject.backgroundColor || '#474555'),
    color: convert.hexColorParse(
        dailyObject.fontColor || '#00C894'),
    lineSpacing: 0,
    padding: 15,
  };
  return convert.textToImageBuffer(dailyMessage, imageOptions);
};
