const sunCalc = require('suncalc');
const { Validator } = require('node-input-validator');

const SunCalc = require('../helpers/sunCalc');

exports.getGreetingText = async (req, res, next) => {
  Date.prototype.now = function () {
    return ((this.getHours() < 10)?"0":"") + this.getHours() +":"+ ((this.getMinutes() < 10)?"0":"") + this.getMinutes() +":"+ ((this.getSeconds() < 10)?"0":"") + this.getSeconds();
  };

  try {
    const timeNow = (
      (new Date().now()).slice(0, 2) < 3 
      ? 24+((new Date().now()).slice(0,2)-3) 
      : (new Date().now()).slice(0,2)-3
    ) + (new Date().now()).slice(2,);
    // TODO convert or accept timezone
    const sunCalculations = sunCalc.getTimes(new Date, -15.7801, -47.9292);
    const sun = {
      set: (sunCalculations.sunset.toISOString().split('T')[1]).split('.')[0],
      rise: (sunCalculations.sunrise.toISOString().split('T')[1]).split('.')[0],
    }
    SunCalc.getCurrentGMTDiffFromDate(-3);
    return res.status(200).json('Good ' + SunCalc.getDayPeriod(sun, timeNow).period);
  } catch (err) {
    return next(err);
  }
};

exports.getGreetingSVG = async (req, res, next) => {
  Date.prototype.now = function () {
    return ((this.getHours() < 10)?"0":"") + this.getHours() +":"+ ((this.getMinutes() < 10)?"0":"") + this.getMinutes() +":"+ ((this.getSeconds() < 10)?"0":"") + this.getSeconds();
  };
  try {
    const timeNow = (
      (new Date().now()).slice(0, 2) < 3 
      ? 24+((new Date().now()).slice(0,2)-3) 
      : (new Date().now()).slice(0,2)-3
    ) + (new Date().now()).slice(2,);
    // TODO convert or accept timezone
    const sunCalculations = sunCalc.getTimes(new Date, -15.7801, -47.9292);
    const sun = {
      set: (sunCalculations.sunset.toISOString().split('T')[1]).split('.')[0],
      rise: (sunCalculations.sunrise.toISOString().split('T')[1]).split('.')[0],
    }
    const dayPeriod = SunCalc.getDayPeriod(sun, timeNow);
    res.setHeader("Content-Type", "image/svg+xml");
    res.setHeader("Cache-control", `public, max-age=${dayPeriod.expires}`);
    res.status(200);
    res.write(
    `<svg
    width="240"
    height="40"
    xmlns="http://www.w3.org/2000/svg"
    >
      <style>
        .header {
          font: 18px Ubuntu;
          fill: #000;
        }
      </style> 
      <g transform="translate(12, 27)">
          <text
            class="header">
            ${dayPeriod.emoji} Good ${dayPeriod.period}
          </text>
      </g>
      ${dayPeriod.text}
    </svg>`);
    return res.end();
  } catch (err) {
    return next(err);
  }
};

/**
  * @function getGreeting
  * 
  * @queryParams {responseType}
  * @param {responseType} string The type o media that will be returned (Text, SVG, ...)
  * @param {latLong} string The latitude and longitude of client separated by comma
*/
exports.getGreeting = async (req, res, next) => {
  let geoLocation = req.query;

  const validateParams = new Validator(geoLocation, {
    responseType: 'required',
    latLong: 'required|latLong',
  });
  const paramsIsValid = await validateParams.check();

  if (!paramsIsValid) {
    return res.status(422).json(validateParams.errors);
  }
  const latitude = geoLocation.latLong.split(',')[0];
  const longitude = geoLocation.latLong.split(',')[1];

  try {
    geoLocation = {
      ...geoLocation,
      timezone: await SunCalc.getTimeZone(latitude, longitude),
      now: new Date(),
      sun: await SunCalc.getSun(latitude, longitude),
    };
    geoLocation = {
      ...geoLocation,
      afternoonStarts: await SunCalc.getAfternoonStarts(geoLocation.sun),
    };
    geoLocation = await SunCalc.getActualPeriod(geoLocation);
    const result = await SunCalc.getResponseByType(geoLocation, res);
    return;
  } catch (err) {
    return res.status(500).json({
      mesage: 'Internal Server Error',
    });
  }
};
