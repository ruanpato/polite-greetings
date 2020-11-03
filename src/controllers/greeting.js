const https = require('https');

const toDate = (string) => Date.parse(`01/01/1990 ${string}`);

const transformAmPmTo24 = (stringAmPm) => {
  const arrayValue = stringAmPm.split(RegExp(':| '));
  return (
    (arrayValue[3] === 'PM' ? 
      +(arrayValue[0] === '12' ? '00' : arrayValue [0])+12
      : (arrayValue[0] === '12' ? '00' : arrayValue [0]))
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
  return ((toDate(timeNow) >= toDate(sun.set)) || (toDate(timeNow) <= sun.rise)) ?
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
    const timeNow = (new Date().now());
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
    const timeNow = (new Date().now());
    // TODO convert or accept timezone
    const sun = await getHttpsResponse('https://api.sunrise-sunset.org/json?lat=-15.7801&lng=-47.9292&date=today');
    const dayPeriod = getDayPeriodEmoji(sun, timeNow);
    res.writeHeader(200, {"Content-Type": "text/html"});  
    res.write(
    `<!DOCTYPE html>
    <html>
    <meta charset="UTF-8">
    <body>
    
    <svg height="30" width="200">
      <text x="0" y="15" fill="black">${dayPeriod.emoji} ${dayPeriod.text}</text>
      ${dayPeriod.text}
    </svg>
     
    </body>
    </html>
    `);
    return res.end();
  } catch (err) {
    return next(err);
  }
};
