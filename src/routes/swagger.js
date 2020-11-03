const express = require('express');
const swaggerUi = require('swagger-ui-express');
const swaggerController = require('../controllers/swagger');

const router = express.Router();

swaggerController.convertYamltoJson();
const swaggerDocument = require('../../docs/politeGreetings.json');

router.use('/', swaggerUi.serve, swaggerUi.setup(swaggerDocument));

module.exports = router;
