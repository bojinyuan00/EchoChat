import { z } from 'zod';

import { idStringSchema } from './common.js';
import { rtpParametersSchema } from './rtp.js';

export const mediaKindSchema = z.enum(['audio', 'video']);

export const createProducerBodySchema = z.object({
  transportId: idStringSchema,
  kind: mediaKindSchema,
  rtpParameters: rtpParametersSchema,
  appData: z.record(z.string(), z.unknown()).optional(),
});

export const producerIdParamSchema = z.object({
  id: idStringSchema,
});

export type CreateProducerBody = z.infer<typeof createProducerBodySchema>;
export type ProducerIdParam = z.infer<typeof producerIdParamSchema>;
export type MediaKind = z.infer<typeof mediaKindSchema>;
