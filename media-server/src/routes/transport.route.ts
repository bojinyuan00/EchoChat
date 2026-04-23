import type { FastifyInstance } from 'fastify';

import {
  connectTransportBodySchema,
  createTransportBodySchema,
  transportIdParamSchema,
} from '../schemas/transport.schema.js';
import {
  closeTransport,
  connectTransport,
  createWebRtcTransport,
} from '../services/transport.service.js';

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

  // Task 16 P2-1：补全 Transport 主动关闭接口，供 Go go-service 在用户离会/断连时
  // 精确清理 orphan transport（避免等待 Router 级联）
  app.delete('/transports/:id', async (request) => {
    const { id } = transportIdParamSchema.parse(request.params);
    closeTransport(id);
    return { ok: true as const };
  });
}
