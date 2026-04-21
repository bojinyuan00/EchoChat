import pino, { type Logger, type LoggerOptions } from 'pino';

import { config } from '../config.js';

function buildLoggerOptions(): LoggerOptions {
  const base: LoggerOptions = {
    level: config.logLevel,
    base: {
      service: 'media-server',
      pid: process.pid,
    },
    timestamp: pino.stdTimeFunctions.isoTime,
    redact: {
      paths: ['req.headers["x-internal-token"]', 'headers["x-internal-token"]'],
      censor: '[REDACTED]',
    },
  };

  if (config.logPretty) {
    base.transport = {
      target: 'pino-pretty',
      options: {
        colorize: true,
        translateTime: 'SYS:HH:MM:ss.l',
        ignore: 'pid,hostname,service',
        singleLine: false,
      },
    };
  }

  return base;
}

export const logger: Logger = pino(buildLoggerOptions());

export function childLogger(bindings: Record<string, unknown>): Logger {
  return logger.child(bindings);
}
