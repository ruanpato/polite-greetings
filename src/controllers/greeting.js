const sunCalc = require('suncalc');

const toSeconds = (timeAsString) => {
  return timeAsString.slice(0,2)*3600
        +timeAsString.slice(3,5)*60
        +timeAsString.slice(6,8)*1;
}

const difference = (firstNumber, secondNumber) => {
  return firstNumber > secondNumber ? 
    firstNumber-secondNumber : secondNumber-firstNumber;
}

const getDayPeriod = (sun, timeNow) => {
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
    return res.status(200).json('Good '+getDayPeriod(sun, timeNow).period);
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
    const dayPeriod = getDayPeriod(sun, timeNow);
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
