const router = require('express').Router();

router.use('/', require('./apiDocs'));
router.use('/greetings', require('./greetings'));

module.exports = router;
