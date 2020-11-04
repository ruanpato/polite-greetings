const express = require('express');

const router = express.Router();

router.use('/greetings', require('./greeting'));

router.use('/', require('./swagger'));

module.exports = router;
