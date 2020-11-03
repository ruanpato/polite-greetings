const express = require('express');
const cors = require('cors');
const bodyParser = require('body-parser');
const log = require('./helpers/lib/log/log');

const router = require('./routes/routes');

const port = process.env.PORT || 3000;
const app = express();

// Middlewares Section
app.use(bodyParser.json());
app.use(cors());

app.use('/', router);

app.listen(port, '0.0.0.0', () => {
  log.info((`Server is listening`), null);
});
