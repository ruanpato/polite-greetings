const NodeQueryBuilder = require('node-querybuilder');

async function getPool() {
  const settings = {
    host: process.env.DB_HOST,
    database: process.env.DB_NAME,
    user: process.env.DB_USER,
    password: process.env.DB_PASS,
  };
  return new NodeQueryBuilder(settings, 'mysql', 'pool');
}

exports.getQueryBuilder = async () => {
  const pool = await getPool();
  return pool.get_connection();
};
