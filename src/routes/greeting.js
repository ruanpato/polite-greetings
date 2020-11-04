const express = require('express');
const greetingController = require('../controllers/greeting');

const router = express.Router();

router.get('/text', greetingController.getGreeting);
router.get('/image', greetingController.getGreetingSVG);

module.exports = router;
