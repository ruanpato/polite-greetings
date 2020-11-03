const express = require('express');
const greetingController = require('../controllers/greeting');

const router = express.Router();

router.get('/', greetingController.getGreeting);

module.exports = router;
