const { json } = require('body-parser');
const express = require('express');
const greetingController = require('../controllers/greeting');

const router = express.Router();

router.get('/text', greetingController.getGreeting);
router.get('/image', greetingController.getGreetingSVG);
router.get('/test', greetingController.getGreetingSVGTest);

module.exports = router;
