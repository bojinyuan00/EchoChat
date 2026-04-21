import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest';

import { closeWorker, startWorker } from '../../src/mediasoup/worker.js';
import {
  closeProducer,
  createProducer,
  getProducer,
  _clearProducerMap,
} from '../../src/services/producer.service.js';
import { _clearRouterMap, createRouter } from '../../src/services/router.service.js';
import {
  _clearTransportMap,
  createWebRtcTransport,
} from '../../src/services/transport.service.js';
import { AppError } from '../../src/utils/errors.js';

let recvTransportId = '';

beforeAll(async () => {
  await startWorker();
  const { routerId } = await createRouter('ROOM-PRODUCER');
  const recv = await createWebRtcTransport({
    routerId,
    userId: 'user-recv',
    direction: 'recv',
  });
  recvTransportId = recv.id;
});

afterAll(async () => {
  _clearProducerMap();
  _clearTransportMap();
  _clearRouterMap();
  await closeWorker();
});

afterEach(() => {
  _clearProducerMap();
});

describe('producer.service', () => {
  it('getProducer throws NOT_FOUND for unknown id', () => {
    try {
      getProducer('non-existent');
      throw new Error('expected throw');
    } catch (err) {
      expect(err).toBeInstanceOf(AppError);
      expect((err as AppError).code).toBe('NOT_FOUND');
    }
  });

  it('closeProducer throws NOT_FOUND for unknown id', async () => {
    await expect(closeProducer('non-existent')).rejects.toMatchObject({ code: 'NOT_FOUND' });
  });

  const validRtpParameters = {
    codecs: [
      { mimeType: 'audio/opus', clockRate: 48000, channels: 2, payloadType: 100 },
    ],
    encodings: [{ ssrc: 111111 }],
  } as never;

  it('createProducer rejects when transport direction is recv', async () => {
    await expect(
      createProducer({
        transportId: recvTransportId,
        kind: 'audio',
        rtpParameters: validRtpParameters,
      }),
    ).rejects.toMatchObject({ code: 'CONFLICT' });
  });

  it('createProducer with unknown transport throws NOT_FOUND', async () => {
    await expect(
      createProducer({
        transportId: 'bogus-transport',
        kind: 'audio',
        rtpParameters: validRtpParameters,
      }),
    ).rejects.toMatchObject({ code: 'NOT_FOUND' });
  });
});
