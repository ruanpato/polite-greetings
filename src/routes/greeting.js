const express = require('express');
const greetingController = require('../controllers/greeting');

const router = express.Router();

router.get('/text', greetingController.getGreetingText);
router.get('/image', greetingController.getGreetingSVG);
router.get('/', greetingController.getGreeting);

module.exports = router;
