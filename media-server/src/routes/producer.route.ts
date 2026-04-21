import type { FastifyInstance } from 'fastify';
import type { RtpParameters } from 'mediasoup/types';

import { createProducerBodySchema, producerIdParamSchema } from '../schemas/producer.schema.js';
import { closeProducer, createProducer } from '../services/producer.service.js';

export async function producerRoutes(app: FastifyInstance): Promise<void> {
  app.post('/producers', async (request, reply) => {
    const body = createProducerBodySchema.parse(request.body);
    const result = await createProducer({
      transportId: body.transportId,
      kind: body.kind,
      rtpParameters: body.rtpParameters as RtpParameters,
      appData: body.appData,
    });
    reply.code(201);
    return result;
  });

  app.delete('/producers/:id', async (request) => {
    const { id } = producerIdParamSchema.parse(request.params);
    await closeProducer(id);
    return { ok: true as const };
  });
}
