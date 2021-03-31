const router = require('express').Router();

const greetingsController = require('../controllers/greetings');

router.get('/', greetingsController.getGreeting);

module.exports = router;
