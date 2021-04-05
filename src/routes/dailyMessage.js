const router = require('express').Router();

const greetingsController = require('../controllers/dailyMessage');

router.get('/', greetingsController.getDailyMessage);

module.exports = router;
