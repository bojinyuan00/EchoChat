import type { FastifyError, FastifyReply, FastifyRequest } from 'fastify';
import { ZodError } from 'zod';

import { AppError } from '../utils/errors.js';

interface ErrorHandlerHost {
  setErrorHandler(
    handler: (error: FastifyError | Error, request: FastifyRequest, reply: FastifyReply) => unknown,
  ): unknown;
}

export function registerErrorHandler(app: ErrorHandlerHost): void {
  app.setErrorHandler((error: FastifyError | Error, request: FastifyRequest, reply: FastifyReply) => {
    if (error instanceof ZodError) {
      const fieldErrors = error.issues.map((issue) => ({
        path: issue.path.join('.') || '(root)',
        code: issue.code,
        message: issue.message,
      }));
      request.log.warn({ fieldErrors, path: request.url }, 'request validation failed');
      return reply.code(400).send({
        code: 'VALIDATION_ERROR',
        message: 'invalid request payload',
        fieldErrors,
      });
    }

    if (error instanceof AppError) {
      request.log.warn(
        { appCode: error.code, details: error.details, path: request.url },
        error.message,
      );
      return reply.code(error.statusCode).send(error.toJSON());
    }

    const fe = error as FastifyError;
    if (fe.statusCode && fe.statusCode >= 400 && fe.statusCode < 500) {
      request.log.warn({ err: fe.message, statusCode: fe.statusCode }, 'client error');
      return reply.code(fe.statusCode).send({
        code: fe.code ?? 'BAD_REQUEST',
        message: fe.message,
      });
    }

    request.log.error(
      {
        err: error.message,
        stack: error.stack,
        path: request.url,
      },
      'unhandled error in request',
    );
    return reply.code(500).send({
      code: 'INTERNAL_ERROR',
      message: 'internal server error',
    });
  });
}
