const express = require('express');
const cors = require('cors');

const log = require('./helpers/lib/log/log');
const router = require('./routes/routes');

require('../util/envLoader');

const protocol = process.env.PROTOCOL || 'http';
const server = process.env.SERVER || 'localhost';
const port = process.env.PORT || 3000;

const app = express();

// Middlewares Section
app.use(express.urlencoded({
  extended: true,
}));
app.use(express.json({
  extended: true,
}));
// Validate Body
app.use(
  (err, req, res, next) => {
    if (err instanceof SyntaxError && err.status === 400 && 'body' in err) {
      return res.status(400).json({
        message: 'Bad Request',
      });
    }
    return next(err);
  },
);
app.use(cors());

app.use('/', router);

app.listen(port, server, () => {
  log.info((`Server is listening at ${protocol}://${server}:${port}`), null);
});
