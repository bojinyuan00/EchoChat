import { describe, expect, it } from 'vitest';

import { AppError, conflict, notFound } from '../src/utils/errors.js';

describe('AppError', () => {
  it('maps NOT_FOUND to 404', () => {
    const err = notFound('router', 'r-1');
    expect(err).toBeInstanceOf(AppError);
    expect(err.code).toBe('NOT_FOUND');
    expect(err.statusCode).toBe(404);
    expect(err.message).toContain('r-1');
    expect(err.toJSON()).toMatchObject({
      code: 'NOT_FOUND',
      message: expect.stringContaining('router'),
      details: { resource: 'router', id: 'r-1' },
    });
  });

  it('maps CONFLICT to 409', () => {
    const err = conflict('already connected');
    expect(err.statusCode).toBe(409);
    expect(err.code).toBe('CONFLICT');
  });

  it('maps ROUTER_LIMIT_EXCEEDED to 503', () => {
    const err = new AppError('ROUTER_LIMIT_EXCEEDED', 'too many routers');
    expect(err.statusCode).toBe(503);
  });

  it('maps CAN_NOT_CONSUME to 400', () => {
    const err = new AppError('CAN_NOT_CONSUME', 'nope');
    expect(err.statusCode).toBe(400);
  });

  it('maps MEDIASOUP_ERROR to 500', () => {
    const err = new AppError('MEDIASOUP_ERROR', 'boom');
    expect(err.statusCode).toBe(500);
  });

  it('toJSON omits details when not provided', () => {
    const err = new AppError('CONFLICT', 'just a conflict');
    expect(err.toJSON()).toEqual({ code: 'CONFLICT', message: 'just a conflict' });
  });
});
