const router = require('express').Router();

router.use('/', require('./apiDocs'));
router.use('/greetings', require('./greetings'));
router.use('/daily-message', require('./dailyMessage'));

module.exports = router;
