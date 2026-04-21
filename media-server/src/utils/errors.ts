export type AppErrorCode =
  | 'NOT_FOUND'
  | 'CONFLICT'
  | 'ROUTER_LIMIT_EXCEEDED'
  | 'CAN_NOT_CONSUME'
  | 'MEDIASOUP_ERROR';

const ERROR_STATUS: Record<AppErrorCode, number> = {
  NOT_FOUND: 404,
  CONFLICT: 409,
  ROUTER_LIMIT_EXCEEDED: 503,
  CAN_NOT_CONSUME: 400,
  MEDIASOUP_ERROR: 500,
};

export class AppError extends Error {
  readonly code: AppErrorCode;
  readonly statusCode: number;
  readonly details: Record<string, unknown> | undefined;

  constructor(code: AppErrorCode, message: string, details?: Record<string, unknown>) {
    super(message);
    this.name = 'AppError';
    this.code = code;
    this.statusCode = ERROR_STATUS[code];
    this.details = details;
  }

  toJSON(): Record<string, unknown> {
    return {
      code: this.code,
      message: this.message,
      ...(this.details ? { details: this.details } : {}),
    };
  }
}

export function notFound(resource: string, id: string): AppError {
  return new AppError('NOT_FOUND', `${resource} not found: ${id}`, { resource, id });
}

export function conflict(message: string, details?: Record<string, unknown>): AppError {
  return new AppError('CONFLICT', message, details);
}
