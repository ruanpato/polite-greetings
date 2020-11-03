const slogger = require('node-slogger');

// TODO create database table to save log, and saves warn and error logs

exports.debug = async (string, object) => {
  slogger.debug(string, object == null ? '' : object);
};

exports.info = async (string, object) => {
  slogger.info(string, object == null ? '' : object);
};

exports.trace = async (string, object) => {
  slogger.trace(string, object == null ? '' : object);
};

exports.warn = async (string, object) => {
  slogger.warn(string, object == null ? '' : object);
};

exports.error = async (string, object) => {
  slogger.error(string, object == null ? '' : object);
};
