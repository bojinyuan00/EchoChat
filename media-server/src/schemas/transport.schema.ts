import { z } from 'zod';

import { idStringSchema, userIdSchema } from './common.js';

export const transportDirectionSchema = z.enum(['send', 'recv']);

export const createTransportBodySchema = z.object({
  routerId: idStringSchema,
  userId: userIdSchema,
  direction: transportDirectionSchema,
});

export const transportIdParamSchema = z.object({
  id: idStringSchema,
});

const dtlsFingerprintSchema = z.object({
  algorithm: z.enum([
    'sha-1',
    'sha-224',
    'sha-256',
    'sha-384',
    'sha-512',
  ]),
  value: z.string().min(1),
});

const dtlsParametersSchema = z.object({
  role: z.enum(['auto', 'client', 'server']).optional(),
  fingerprints: z.array(dtlsFingerprintSchema).min(1),
});

export const connectTransportBodySchema = z.object({
  dtlsParameters: dtlsParametersSchema,
});

export type CreateTransportBody = z.infer<typeof createTransportBodySchema>;
export type TransportIdParam = z.infer<typeof transportIdParamSchema>;
export type ConnectTransportBody = z.infer<typeof connectTransportBodySchema>;
export type TransportDirection = z.infer<typeof transportDirectionSchema>;
