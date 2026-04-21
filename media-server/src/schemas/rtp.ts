import { z } from 'zod';

/**
 * 对 mediasoup 的 RtpParameters / RtpCapabilities 做"浅层存在性校验"。
 *
 * 不校验完整结构（交给 mediasoup native 层做严格校验），但至少保证：
 *   - 必填字段存在
 *   - codecs 至少 1 项
 *   - 客户端传入 {} / null / 空数组 等明显错误 → 返回 400 VALIDATION_ERROR
 *
 * 通过 .passthrough() 保留未知字段向后兼容。
 */

const rtcpFeedbackSchema = z
  .object({
    type: z.string(),
    parameter: z.string().optional(),
  })
  .passthrough();

const baseCodecSchema = z
  .object({
    mimeType: z.string().min(1),
    clockRate: z.number().int().positive(),
    channels: z.number().int().positive().optional(),
    parameters: z.record(z.string(), z.unknown()).optional(),
    rtcpFeedback: z.array(rtcpFeedbackSchema).optional(),
  })
  .passthrough();

const rtpParametersCodecSchema = baseCodecSchema.extend({
  payloadType: z.number().int().min(0).max(255),
});

const rtpCapabilitiesCodecSchema = baseCodecSchema.extend({
  kind: z.enum(['audio', 'video']).optional(),
  preferredPayloadType: z.number().int().min(0).max(255).optional(),
});

const rtpHeaderExtensionParameterSchema = z
  .object({
    uri: z.string().min(1),
    id: z.number().int().min(0),
    encrypt: z.boolean().optional(),
    parameters: z.record(z.string(), z.unknown()).optional(),
  })
  .passthrough();

const rtpEncodingParametersSchema = z
  .object({
    ssrc: z.number().int().nonnegative().optional(),
    rid: z.string().optional(),
    scalabilityMode: z.string().optional(),
    maxBitrate: z.number().int().nonnegative().optional(),
  })
  .passthrough();

export const rtpParametersSchema = z
  .object({
    mid: z.string().optional(),
    codecs: z.array(rtpParametersCodecSchema).min(1, 'rtpParameters.codecs must not be empty'),
    headerExtensions: z.array(rtpHeaderExtensionParameterSchema).optional(),
    encodings: z.array(rtpEncodingParametersSchema).optional(),
    rtcp: z
      .object({
        cname: z.string().optional(),
        reducedSize: z.boolean().optional(),
      })
      .passthrough()
      .optional(),
  })
  .passthrough();

export const rtpCapabilitiesSchema = z
  .object({
    codecs: z
      .array(rtpCapabilitiesCodecSchema)
      .min(1, 'rtpCapabilities.codecs must not be empty'),
    headerExtensions: z.array(z.unknown()).optional(),
  })
  .passthrough();

export type RtpParametersInput = z.infer<typeof rtpParametersSchema>;
export type RtpCapabilitiesInput = z.infer<typeof rtpCapabilitiesSchema>;
