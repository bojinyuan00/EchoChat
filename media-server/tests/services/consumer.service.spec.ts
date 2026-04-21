import { afterAll, beforeAll, describe, expect, it } from 'vitest';

import { closeWorker, startWorker } from '../../src/mediasoup/worker.js';
import {
  closeConsumer,
  createConsumer,
  resumeConsumer,
  _clearConsumerMap,
} from '../../src/services/consumer.service.js';
import { _clearProducerMap } from '../../src/services/producer.service.js';
import { _clearRouterMap, createRouter } from '../../src/services/router.service.js';
import {
  _clearTransportMap,
  createWebRtcTransport,
} from '../../src/services/transport.service.js';
import { AppError } from '../../src/utils/errors.js';

let routerId = '';
let sendTransportId = '';
let recvTransportId = '';

beforeAll(async () => {
  await startWorker();
  const r = await createRouter('ROOM-CONSUMER');
  routerId = r.routerId;
  sendTransportId = (
    await createWebRtcTransport({ routerId, userId: 'u-s', direction: 'send' })
  ).id;
  recvTransportId = (
    await createWebRtcTransport({ routerId, userId: 'u-r', direction: 'recv' })
  ).id;
});

afterAll(async () => {
  _clearConsumerMap();
  _clearProducerMap();
  _clearTransportMap();
  _clearRouterMap();
  await closeWorker();
});

describe('consumer.service', () => {
  it('createConsumer with unknown router throws NOT_FOUND', async () => {
    await expect(
      createConsumer({
        routerId: 'bogus-router',
        transportId: recvTransportId,
        producerId: 'bogus-producer',
        rtpCapabilities: {} as never,
      }),
    ).rejects.toMatchObject({ code: 'NOT_FOUND' });
  });

  it('createConsumer with unknown producer throws NOT_FOUND', async () => {
    await expect(
      createConsumer({
        routerId,
        transportId: recvTransportId,
        producerId: 'bogus-producer',
        rtpCapabilities: {} as never,
      }),
    ).rejects.toMatchObject({ code: 'NOT_FOUND' });
  });

  it('createConsumer with unknown transport throws NOT_FOUND', async () => {
    await expect(
      createConsumer({
        routerId,
        transportId: 'bogus-transport',
        producerId: 'bogus-producer',
        rtpCapabilities: {} as never,
      }),
    ).rejects.toMatchObject({ code: 'NOT_FOUND' });
  });

  it('resumeConsumer throws NOT_FOUND for unknown id', async () => {
    await expect(resumeConsumer('non-existent')).rejects.toMatchObject({ code: 'NOT_FOUND' });
  });

  it('closeConsumer throws NOT_FOUND for unknown id', async () => {
    await expect(closeConsumer('non-existent')).rejects.toMatchObject({ code: 'NOT_FOUND' });
  });

  it('placeholder: send transport exists for coverage', () => {
    expect(sendTransportId).toBeTruthy();
  });
});
