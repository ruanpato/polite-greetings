const router = require('express').Router();
const swaggerUi = require('swagger-ui-express');

const apiDocsController = require('../controllers/apiDocs');

router.use(
    '/api-docs',
    swaggerUi.serve,
    swaggerUi.setup(apiDocsController.getSwaggerDocument()),
);

module.exports = router;
