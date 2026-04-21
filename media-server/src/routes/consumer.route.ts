import type { FastifyInstance } from 'fastify';
import type { RtpCapabilities } from 'mediasoup/types';

import { consumerIdParamSchema, createConsumerBodySchema } from '../schemas/consumer.schema.js';
import { closeConsumer, createConsumer, resumeConsumer } from '../services/consumer.service.js';

export async function consumerRoutes(app: FastifyInstance): Promise<void> {
  app.post('/consumers', async (request, reply) => {
    const body = createConsumerBodySchema.parse(request.body);
    const result = await createConsumer({
      routerId: body.routerId,
      transportId: body.transportId,
      producerId: body.producerId,
      rtpCapabilities: body.rtpCapabilities as RtpCapabilities,
    });
    reply.code(201);
    return result;
  });

  app.post('/consumers/:id/resume', async (request) => {
    const { id } = consumerIdParamSchema.parse(request.params);
    await resumeConsumer(id);
    return { ok: true as const };
  });

  app.delete('/consumers/:id', async (request) => {
    const { id } = consumerIdParamSchema.parse(request.params);
    await closeConsumer(id);
    return { ok: true as const };
  });
}
