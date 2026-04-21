import { z } from 'zod';

export const idStringSchema = z
  .string()
  .min(1)
  .max(128)
  .regex(/^[A-Za-z0-9:_-]+$/u, 'id must match [A-Za-z0-9:_-]+');

export const roomCodeSchema = z
  .string()
  .min(3)
  .max(64)
  .regex(/^[A-Z0-9-]+$/u, 'roomCode must match [A-Z0-9-]+ (uppercased)');

export const userIdSchema = z
  .union([z.string().min(1).max(64), z.number().int().positive()])
  .transform((val) => String(val));

export const okResponseSchema = z.object({
  ok: z.literal(true),
});

export type OkResponse = z.infer<typeof okResponseSchema>;
