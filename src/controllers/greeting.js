const https = require('https');
const sunCalc = require('suncalc');

const toDate = (string) => Date.parse(`01/01/1990 ${string}`);

const getDayPeriod = (sun, timeNow) => {
  return ((toDate(timeNow) >= toDate(sun.set)) && (toDate(timeNow) <= sun.rise)) ?
      'Boa Noite'
      : (toDate(timeNow) > toDate('12:00:00') ?
        'Boa Tarde'
        : 'Bom dia'
      )
};

const getDayPeriodEmoji = (sun, timeNow) => {
  return ((toDate(timeNow) >= toDate(sun.set)) || (toDate(timeNow) <= sun.rise)) ?
      {emoji: '🌕', text: 'Boa Noite'}
      : (toDate(timeNow) > toDate('12:00:00') ?
        {emoji: '☀️', text: 'Boa Tarde'}
        : {emoji: '🌅', text: 'Bom dia'}
      )
};

exports.getGreeting = async (req, res, next) => {
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
    return res.status(200).json(getDayPeriod(sun, timeNow));
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
    const dayPeriod = getDayPeriodEmoji(sun, timeNow);
    res.setHeader("Content-Type", "image/svg+xml",);
    res.setHeader("Cache-control", "no-cache");
    res.status(200);
    res.write(
    `<svg
    width="140"
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
            ${dayPeriod.emoji} ${dayPeriod.text}
          </text>
      </g>
      ${dayPeriod.text}
    </svg>`);
    return res.end();
  } catch (err) {
    return next(err);
  }
};
