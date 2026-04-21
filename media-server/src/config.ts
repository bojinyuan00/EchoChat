import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { z } from 'zod';

function loadDotEnv(): void {
  const envPath = resolve(process.cwd(), '.env');
  if (!existsSync(envPath)) {
    return;
  }
  const raw = readFileSync(envPath, 'utf-8');
  for (const line of raw.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) {
      continue;
    }
    const eq = trimmed.indexOf('=');
    if (eq <= 0) {
      continue;
    }
    const key = trimmed.slice(0, eq).trim();
    let value = trimmed.slice(eq + 1).trim();
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1);
    }
    if (process.env[key] === undefined) {
      process.env[key] = value;
    }
  }
}

loadDotEnv();

const booleanSchema = z
  .union([z.string(), z.boolean()])
  .transform((val) => {
    if (typeof val === 'boolean') return val;
    const lowered = val.toLowerCase();
    return lowered === '1' || lowered === 'true' || lowered === 'yes' || lowered === 'on';
  });

const envSchema = z
  .object({
    NODE_ENV: z.enum(['development', 'production', 'test']).default('development'),

    HTTP_HOST: z.string().min(1).default('0.0.0.0'),
    HTTP_PORT: z.coerce.number().int().min(1).max(65535).default(3300),

    LOG_LEVEL: z
      .enum(['trace', 'debug', 'info', 'warn', 'error', 'fatal', 'silent'])
      .default('info'),
    LOG_PRETTY: booleanSchema.default('true'),

    MEDIA_INTERNAL_TOKEN: z
      .string()
      .min(8, 'MEDIA_INTERNAL_TOKEN must be at least 8 characters'),

    MEDIASOUP_LISTEN_IP: z.string().min(1).default('0.0.0.0'),
    MEDIASOUP_ANNOUNCED_IP: z
      .string()
      .optional()
      .transform((v) => (v && v.trim() !== '' ? v.trim() : undefined)),

    MEDIASOUP_RTC_MIN_PORT: z.coerce.number().int().min(1024).max(65535).default(40000),
    MEDIASOUP_RTC_MAX_PORT: z.coerce.number().int().min(1024).max(65535).default(40199),

    MEDIASOUP_WORKER_LOG_LEVEL: z
      .enum(['debug', 'warn', 'error', 'none'])
      .default('warn'),

    MEDIASOUP_MAX_ROUTERS: z.coerce.number().int().min(1).default(200),
  })
  .superRefine((data, ctx) => {
    if (data.MEDIASOUP_RTC_MIN_PORT >= data.MEDIASOUP_RTC_MAX_PORT) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['MEDIASOUP_RTC_MAX_PORT'],
        message: 'MEDIASOUP_RTC_MAX_PORT must be greater than MEDIASOUP_RTC_MIN_PORT',
      });
    }
  });

function parseEnv() {
  const parsed = envSchema.safeParse(process.env);
  if (!parsed.success) {
    const issues = parsed.error.issues
      .map((issue) => `  - ${issue.path.join('.')}: ${issue.message}`)
      .join('\n');
    // eslint-disable-next-line no-console
    console.error(`[media-server] Invalid environment configuration:\n${issues}`);
    process.exit(1);
  }
  return parsed.data;
}

const env = parseEnv();

export const config = {
  nodeEnv: env.NODE_ENV,
  isProduction: env.NODE_ENV === 'production',

  http: {
    host: env.HTTP_HOST,
    port: env.HTTP_PORT,
  },

  logLevel: env.LOG_LEVEL,
  logPretty: env.LOG_PRETTY,

  internalToken: env.MEDIA_INTERNAL_TOKEN,

  mediasoup: {
    listenIp: env.MEDIASOUP_LISTEN_IP,
    announcedIp: env.MEDIASOUP_ANNOUNCED_IP,
    rtcMinPort: env.MEDIASOUP_RTC_MIN_PORT,
    rtcMaxPort: env.MEDIASOUP_RTC_MAX_PORT,
    workerLogLevel: env.MEDIASOUP_WORKER_LOG_LEVEL,
    maxRouters: env.MEDIASOUP_MAX_ROUTERS,
  },
} as const;

export type AppConfig = typeof config;
