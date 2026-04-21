import { z } from 'zod';

import { idStringSchema, roomCodeSchema } from './common.js';

export const createRouterBodySchema = z.object({
  roomCode: roomCodeSchema,
});

export const routerIdParamSchema = z.object({
  routerId: idStringSchema,
});

export type CreateRouterBody = z.infer<typeof createRouterBodySchema>;
export type RouterIdParam = z.infer<typeof routerIdParamSchema>;
