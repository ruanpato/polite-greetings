const GeoTz = require('geo-tz');
const SunCalc = require('suncalc');


const difference = (firstNumber, secondNumber) => {
  return firstNumber > secondNumber ?
    firstNumber-secondNumber : secondNumber-firstNumber;
};

// Translate in the future
exports.getGreeting = async (dayPeriod) => {
  return `Good ${dayPeriod}`;
};

exports.getTimeZone = async (latitude, longitude) =>
  (GeoTz(latitude, longitude)[0]);

exports.getSun = async (latitude, longitude) => {
  const today = new Date();
  const times = SunCalc.getTimes(today, latitude, longitude);

  const tomorrow = new Date(today);
  tomorrow.setDate(tomorrow.getDate() + 1);
  const timesTomorrow = SunCalc.getTimes(tomorrow, latitude, longitude);

  return {
    rise: times.sunrise,
    set: times.sunset,
    riseTomorrow: timesTomorrow.sunrise,
  };
};

exports.getAfternoonStarts = async (sun) => {
  const afternoon =
    new Date(`${sun.rise.toISOString().split('T')[0]}T12:00:00.000Z`);
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
    return {
      ...geoLocation,
      expiresIn: Math.ceil(difference(period.now, period.afternoonStart)/1000),
      period: 'Morning',
      emoji: '🌅',
    };
  }
  if (period.now >= period.afternoonStart && period.now < period.nightStart) {
    return {
      ...geoLocation,
      expiresIn: Math.ceil(difference(period.now, period.nightStart)/1000),
      period: 'Afternoon',
      emoji: '☀️',
    };
  }

  return {
    ...geoLocation,
    expiresIn: Math.ceil(difference(period.now, period.riseTomorrow)/1000),
    period: 'Night',
    emoji: '🌙',
  };
};
