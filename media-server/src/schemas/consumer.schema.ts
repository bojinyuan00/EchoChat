import { z } from 'zod';

import { idStringSchema } from './common.js';
import { rtpCapabilitiesSchema } from './rtp.js';

export const createConsumerBodySchema = z.object({
  routerId: idStringSchema,
  transportId: idStringSchema,
  producerId: idStringSchema,
  rtpCapabilities: rtpCapabilitiesSchema,
});

export const consumerIdParamSchema = z.object({
  id: idStringSchema,
});

export type CreateConsumerBody = z.infer<typeof createConsumerBodySchema>;
export type ConsumerIdParam = z.infer<typeof consumerIdParamSchema>;
