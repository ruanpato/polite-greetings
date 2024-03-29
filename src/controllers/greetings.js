const {Validator} = require('node-input-validator');

const log = require('../helpers/lib/log/log');
const greetingsHelper = require('../helpers/greetings');
const convert = require('../helpers/conversion');

const mountGreetingObject = async (latitude, longitude, geoLocation) => {
  let greetingObject = {
    ...geoLocation,
    timezone: await greetingsHelper.getTimeZone(latitude, longitude),
    now: new Date(),
    sun: await greetingsHelper.getSun(latitude, longitude),
  };
  greetingObject.afternoonStarts = await greetingsHelper.getAfternoonStarts(
      greetingObject,
  );
  greetingObject = await greetingsHelper.getActualPeriod(greetingObject);
  greetingObject.message = greetingObject.emoji + ' ' +
    await greetingsHelper.getGreeting(
        greetingObject.period,
    );
  // Default don't show timezone
  greetingObject.showTz = greetingObject.showTz ? greetingObject.showTz : false;
  return greetingObject;
};

const mountResponse = async (greetingObject, res) => {
  res.set({
    'Cache-control': `public, max-age=${greetingObject.expiresIn}`,
  });
  let returnText = greetingObject.message;
  if (convert.stringAsBool(greetingObject.showTz)) {
    returnText += `\n✈ ${greetingObject.timezone}`;
  }
  if (greetingObject.responseType.toLowerCase() === 'png') {
    const imageOptions = {
      font: convert.fontSizeParse(greetingObject.fontSize),
      backgroundColor: convert.hexColorParse(
          greetingObject.backgroundColor || '#474555'),
      color: convert.hexColorParse(
          greetingObject.fontColor || '#00C894'),
      lineSpacing: 10,
      padding: 15,
    };
    const image = convert.
        textToImageBuffer(returnText, imageOptions);
    res.set(image.header);
    return res.status(200).end(image.buffer);
  }
  if (greetingObject.responseType.toLowerCase() === 'text') {
    res.set({'Content-Type': 'text/plain'});
    return res.status(200).end(returnText);
  }
  res.set({'Content-Type': 'application/json'});
  return res.status(200).json(greetingObject);
};

exports.getGreeting = async (req, res, next) => {
  const geoLocation = req.query;

  const validateParams = new Validator(geoLocation, {
    responseType: 'string',
    latLong: 'required|latLong',
    showTz: 'boolean',
    fontColor: 'hexColor',
    fontSize: 'integer',
    backgroundColor: 'hexColor',
  });
  const paramsIsValid = await validateParams.check();

  if (!paramsIsValid) {
    return res.status(422).json(validateParams.errors);
  }
  const latitude = geoLocation.latLong.split(',')[0];
  const longitude = geoLocation.latLong.split(',')[1];
  try {
    const greetingObject = await mountGreetingObject(
        latitude,
        longitude,
        geoLocation,
    );
    res = await mountResponse(greetingObject, res);
    return;
  } catch (err) {
    log.error('Error:  ', err);
    return res.status(500).json({
      mesage: 'Internal Server Error',
    });
  }
};
