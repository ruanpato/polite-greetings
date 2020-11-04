const express = require('express');

const router = express.Router();

router.use('/tst', require('./greeting'));

router.use('/', require('./swagger'));

module.exports = router;
