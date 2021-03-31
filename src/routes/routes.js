const express = require('express');

const router = express.Router();

router.use('/greetings', require('./greeting'));

router.use('/', require('./apiDocs'));

module.exports = router;
