const express = require('express');
const cors = require('cors');

const log = require('./helpers/lib/log/log');
const router = require('./routes/routes');

require('../util/envLoader');

const configureServer = () => {
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

  app.use('/api/v1', router);

  return app;
};

exports.startServer = () => {
  const port = process.env.PORT || 5000;
  const host = process.env.SERVER || '0.0.0.0';
  const server = configureServer();
  return server.listen(port, host, () => {
    log.info((`Server is listening at ${host}:${port}`), null);
  });
};
