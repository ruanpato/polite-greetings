const GeoTz = require('geo-tz');
const SunCalc = require('suncalc');

const toSeconds = (timeAsString) => {
  return timeAsString.slice(0,2)*3600
    +timeAsString.slice(3,5)*60
    +timeAsString.slice(6,8)*1;
}
  
const difference = (firstNumber, secondNumber) => {
  return firstNumber > secondNumber ? 
    firstNumber-secondNumber : secondNumber-firstNumber;
}

exports.getDayPeriod = (sun, timeNow) => {
  if (toSeconds(timeNow) >= toSeconds(sun.set) || (toSeconds(timeNow) <= sun.rise))
    return {
      emoji: '🌕',
      period: 'Night',
      expires: difference(toSeconds('12:00:00'), toSeconds(timeNow))
    };
  else {
    if (toSeconds(timeNow) > toSeconds('12:00:00'))
      return {
        emoji: '☀️', 
        period: 'Afternoon', 
        expires: difference(toSeconds(sun.set), toSeconds(timeNow))
      };
    // const {sunrise} = sunCalc.getTimes(today.setDate(today.getDate()+1), -15.7801, -47.9292);
    return {
      emoji: '🌅',
      period: 'Morning',
      expires: '300'
    };
  }
}

exports.getTimeZone = async (latitude, longitude) => (GeoTz(latitude, longitude)[0]);

exports.getSun = async (latitude, longitude) => {
  const today = new Date();
  const times = SunCalc.getTimes(today, latitude, longitude);

  const tomorrow = new Date(today);
  tomorrow.setDate(tomorrow.getDate() + 1);
  const timesTomorrow = SunCalc.getTimes(tomorrow, latitude, longitude);

  return {
    rise: times.sunrise,
    set:  times.sunset,
    riseTomorrow: timesTomorrow.sunrise,
  };
};

exports.getAfternoonStarts = async (sun) => {
  const afternoon = new Date(`${sun.rise.toISOString().split('T')[0]}T12:00:00.000Z`);
  return afternoon;
};

exports.getActualPeriod = async (geoLocation) => {
  const period = {
    now: (new Date(geoLocation.now)).getTime(),
    morningStart: (new Date(geoLocation.sun.rise)).getTime(),
    afternoonStart: (new Date(geoLocation.afternoonStarts)).getTime(),
    nightStart: (new Date(geoLocation.sun.set)).getTime(),
    riseTomorrow: (new Date(geoLocation.sun.riseTomorrow)).getTime(),
  };
  if (period.now >= period.morningStart && period.now < period.afternoonStart) {
    geoLocation.period = 'Morning'
    geoLocation.emoji = '🌅'
    geoLocation.expiresIn = Math.ceil(difference(period.now, period.afternoonStart)/1000);
  } else if (period.now >= period.afternoonStart && period.now < period.nightStart) {
    geoLocation.expiresIn = Math.ceil(difference(period.now, period.nightStart)/1000);
    geoLocation.period = 'Afternoon'
    geoLocation.emoji = '☀️'
  } else {
    geoLocation.expiresIn = Math.ceil(difference(period.now, period.riseTomorrow)/1000);
    geoLocation.period = 'Night';
    geoLocation.emoji = '🌕'
  }
  return geoLocation;
};

exports.getResponseByType = async(geoLocation, res) => {
  console.log(geoLocation);
  if (geoLocation.responseType.toLowerCase() === 'text') {
    res.set( {
      'Content-Type': 'text/plain',
      'Cache-control': `public, max-age=${geoLocation.expiresIn}`,
    });
    res.status(200);
    const response = await getGreeting(geoLocation.period);
    res.end(response.toString());
    return;
  }
  if (geoLocation.responseType.toLowerCase() === 'json') {
    res.status(200);
    res.set( {
      'Content-Type': 'application/json',
      'Cache-control': `public, max-age=${geoLocation.expiresIn}`,
    });

    const response = {
      ...geoLocation,
      message: await getGreeting(geoLocation.period),
    };
    res.json(response);
    return;
  }
  if (geoLocation.responseType.toLowerCase() === 'svg') {
    res.status(200);
    res.set( {
      'Content-Type': 'image/svg+xml',
      'Cache-control': `public, max-age=${geoLocation.expiresIn}`,
    });
    const response = {
      ...geoLocation,
      message: await getGreeting(geoLocation.period),
    };
    const svg = await createSvg(response);
    res.write(svg);
    return;
  }
};

// Translate in the future
const getGreeting = async (dayPeriod) => {
  return `Good ${dayPeriod}`;
};

const createSvg = async (geoLocation) => {
  return `<svg width="275" height="60" xmlns="http://www.w3.org/2000/svg">
    <g>
      <rect x="0" y="0" width="340" height="80" fill="#2C302E"></rect>
      <text x="10" y="40" font-family="Verdana" font-size="27" fill="#fff">${geoLocation.emoji} ${geoLocation.message}</text>
    </g>
 </svg>`;
};
