const express = require('express');
const greetingController = require('../controllers/greeting');

const router = express.Router();

router.get('/', greetingController.getGreeting);
router.get('/svg', greetingController.getGreetingSVG);

module.exports = router;
