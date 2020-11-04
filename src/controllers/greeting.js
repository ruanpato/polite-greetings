const { time } = require('console');
const https = require('https');
const { type } = require('os');

const toDate = (string) => Date.parse(`01/01/1990 ${string}`);

const transformAmPmTo24 = (stringAmPm) => {
  const arrayValue = stringAmPm.split(RegExp(':| '));
  return (
    (arrayValue[3] === 'PM' ? 
      +(arrayValue[0] === '12' ? '00' : arrayValue [0])+12
      : +(arrayValue[0] === '12' ? '00' : arrayValue [0]))
    +':'+arrayValue[1]
    +':'+arrayValue[2]
  );
};

const getHttpsResponse = (url) => new Promise((resolve, reject) => {
  https.get(url, res => {
    res.setEncoding('utf8');
    let body = '';
    res.on('data', chunk => (body += chunk));
    res.on('end', () => resolve(
      {
        set: transformAmPmTo24(JSON.parse(body).results.sunset),
        rise: transformAmPmTo24(JSON.parse(body).results.sunrise),
      },
    ));
  }).on('error', reject);
});

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
    const sun = await getHttpsResponse('https://api.sunrise-sunset.org/json?lat=-15.7801&lng=-47.9292&date=today');
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
    console.log(timeNow);
    // TODO convert or accept timezone
    const sun = await getHttpsResponse('https://api.sunrise-sunset.org/json?lat=-15.7801&lng=-47.9292&date=today');
    const dayPeriod = getDayPeriodEmoji(sun, timeNow);
    res.setHeader("Cache-Control", `public, max-age=300`);
    res.setHeader("Cache-Control", `public, max-age=300`);
    res.setHeader("Content-Type", "image/svg+xml");
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
