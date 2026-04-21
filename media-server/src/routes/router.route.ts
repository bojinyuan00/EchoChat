import type { FastifyInstance } from 'fastify';

import { createRouterBodySchema, routerIdParamSchema } from '../schemas/router.schema.js';
import { closeRouter, createRouter } from '../services/router.service.js';

export async function routerRoutes(app: FastifyInstance): Promise<void> {
  app.post('/routers', async (request, reply) => {
    const { roomCode } = createRouterBodySchema.parse(request.body);
    const result = await createRouter(roomCode);
    reply.code(201);
    return result;
  });

  app.delete('/routers/:routerId', async (request) => {
    const { routerId } = routerIdParamSchema.parse(request.params);
    await closeRouter(routerId);
    return { ok: true as const };
  });
}
