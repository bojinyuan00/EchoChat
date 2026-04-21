import type { FastifyInstance } from 'fastify';

import {
  connectTransportBodySchema,
  createTransportBodySchema,
  transportIdParamSchema,
} from '../schemas/transport.schema.js';
import { connectTransport, createWebRtcTransport } from '../services/transport.service.js';

export async function transportRoutes(app: FastifyInstance): Promise<void> {
  app.post('/transports', async (request, reply) => {
    const body = createTransportBodySchema.parse(request.body);
    const result = await createWebRtcTransport({
      routerId: body.routerId,
      userId: body.userId,
      direction: body.direction,
    });
    reply.code(201);
    return result;
  });

  app.post('/transports/:id/connect', async (request) => {
    const { id } = transportIdParamSchema.parse(request.params);
    const { dtlsParameters } = connectTransportBodySchema.parse(request.body);
    await connectTransport({ transportId: id, dtlsParameters });
    return { ok: true as const };
  });
}
