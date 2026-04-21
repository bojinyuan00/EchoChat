import { describe, expect, it } from 'vitest';

import {
  idStringSchema,
  roomCodeSchema,
  userIdSchema,
} from '../src/schemas/common.js';
import { createConsumerBodySchema } from '../src/schemas/consumer.schema.js';
import { createProducerBodySchema } from '../src/schemas/producer.schema.js';
import { createRouterBodySchema } from '../src/schemas/router.schema.js';
import {
  connectTransportBodySchema,
  createTransportBodySchema,
} from '../src/schemas/transport.schema.js';

describe('common schemas', () => {
  it('accepts valid id strings', () => {
    expect(idStringSchema.parse('abc-123_X:Y')).toBe('abc-123_X:Y');
  });

  it('rejects invalid characters in id', () => {
    expect(() => idStringSchema.parse('abc/def')).toThrow();
    expect(() => idStringSchema.parse('')).toThrow();
    expect(() => idStringSchema.parse('a'.repeat(200))).toThrow();
  });

  it('accepts valid room codes', () => {
    expect(roomCodeSchema.parse('ROOM-001')).toBe('ROOM-001');
  });

  it('rejects lowercase or invalid room codes', () => {
    expect(() => roomCodeSchema.parse('room-001')).toThrow();
    expect(() => roomCodeSchema.parse('AB')).toThrow();
  });

  it('coerces numeric userIds into strings', () => {
    expect(userIdSchema.parse(42)).toBe('42');
    expect(userIdSchema.parse('alice')).toBe('alice');
  });

  it('rejects non-positive numeric userIds', () => {
    expect(() => userIdSchema.parse(0)).toThrow();
    expect(() => userIdSchema.parse(-1)).toThrow();
  });
});

describe('router schema', () => {
  it('parses valid body', () => {
    expect(createRouterBodySchema.parse({ roomCode: 'ROOM-1' })).toEqual({
      roomCode: 'ROOM-1',
    });
  });

  it('rejects missing roomCode', () => {
    expect(() => createRouterBodySchema.parse({})).toThrow();
  });
});

describe('transport schemas', () => {
  it('parses create body', () => {
    const parsed = createTransportBodySchema.parse({
      routerId: 'router-1',
      userId: 'user-1',
      direction: 'send',
    });
    expect(parsed.direction).toBe('send');
  });

  it('rejects bad direction', () => {
    expect(() =>
      createTransportBodySchema.parse({
        routerId: 'router-1',
        userId: 'user-1',
        direction: 'bidirectional',
      }),
    ).toThrow();
  });

  it('parses connect body with fingerprints', () => {
    const parsed = connectTransportBodySchema.parse({
      dtlsParameters: {
        role: 'auto',
        fingerprints: [{ algorithm: 'sha-256', value: 'AA:BB' }],
      },
    });
    expect(parsed.dtlsParameters.fingerprints).toHaveLength(1);
  });

  it('rejects connect body without fingerprints', () => {
    expect(() =>
      connectTransportBodySchema.parse({
        dtlsParameters: { role: 'auto', fingerprints: [] },
      }),
    ).toThrow();
  });
});

describe('producer schema', () => {
  const validRtpParameters = {
    codecs: [
      { mimeType: 'audio/opus', clockRate: 48000, payloadType: 100, channels: 2 },
    ],
  };

  it('parses valid body', () => {
    const parsed = createProducerBodySchema.parse({
      transportId: 'trans-1',
      kind: 'audio',
      rtpParameters: validRtpParameters,
    });
    expect(parsed.kind).toBe('audio');
    expect(parsed.rtpParameters.codecs[0]?.payloadType).toBe(100);
  });

  it('rejects bad kind', () => {
    expect(() =>
      createProducerBodySchema.parse({
        transportId: 'trans-1',
        kind: 'data',
        rtpParameters: validRtpParameters,
      }),
    ).toThrow();
  });

  it('rejects rtpParameters without codecs', () => {
    expect(() =>
      createProducerBodySchema.parse({
        transportId: 'trans-1',
        kind: 'audio',
        rtpParameters: {},
      }),
    ).toThrow();
  });

  it('rejects rtpParameters with empty codecs', () => {
    expect(() =>
      createProducerBodySchema.parse({
        transportId: 'trans-1',
        kind: 'audio',
        rtpParameters: { codecs: [] },
      }),
    ).toThrow();
  });
});

describe('consumer schema', () => {
  const validRtpCapabilities = {
    codecs: [{ mimeType: 'audio/opus', clockRate: 48000 }],
  };

  it('parses valid body', () => {
    const parsed = createConsumerBodySchema.parse({
      routerId: 'r-1',
      transportId: 't-1',
      producerId: 'p-1',
      rtpCapabilities: validRtpCapabilities,
    });
    expect(parsed.routerId).toBe('r-1');
  });

  it('rejects missing fields', () => {
    expect(() =>
      createConsumerBodySchema.parse({ routerId: 'r-1' }),
    ).toThrow();
  });

  it('rejects rtpCapabilities with empty codecs', () => {
    expect(() =>
      createConsumerBodySchema.parse({
        routerId: 'r-1',
        transportId: 't-1',
        producerId: 'p-1',
        rtpCapabilities: { codecs: [] },
      }),
    ).toThrow();
  });
});
