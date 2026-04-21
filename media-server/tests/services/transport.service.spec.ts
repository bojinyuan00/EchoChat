import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest';

import { closeWorker, startWorker } from '../../src/mediasoup/worker.js';
import { _clearRouterMap, createRouter } from '../../src/services/router.service.js';
import {
  _clearTransportMap,
  createWebRtcTransport,
  getTransport,
  getTransportStats,
} from '../../src/services/transport.service.js';
import { AppError } from '../../src/utils/errors.js';

let routerId = '';

beforeAll(async () => {
  await startWorker();
  const r = await createRouter('ROOM-TRANSPORT');
  routerId = r.routerId;
});

afterAll(async () => {
  _clearTransportMap();
  _clearRouterMap();
  await closeWorker();
});

afterEach(() => {
  _clearTransportMap();
});

describe('transport.service', () => {
  it('creates a WebRTC transport and returns ICE/DTLS params', async () => {
    const transport = await createWebRtcTransport({
      routerId,
      userId: 'user-1',
      direction: 'send',
    });
    expect(transport.id).toBeTruthy();
    expect(transport.iceParameters.usernameFragment).toBeTruthy();
    expect(transport.iceParameters.password).toBeTruthy();
    expect(Array.isArray(transport.iceCandidates)).toBe(true);
    expect(transport.iceCandidates.length).toBeGreaterThan(0);
    expect(transport.dtlsParameters.fingerprints.length).toBeGreaterThan(0);
    expect(getTransportStats().total).toBe(1);
  });

  it('supports recv direction', async () => {
    const transport = await createWebRtcTransport({
      routerId,
      userId: 'user-2',
      direction: 'recv',
    });
    expect(transport.id).toBeTruthy();
  });

  it('getTransport throws NOT_FOUND for unknown id', () => {
    try {
      getTransport('non-existent');
      throw new Error('expected throw');
    } catch (err) {
      expect(err).toBeInstanceOf(AppError);
      expect((err as AppError).code).toBe('NOT_FOUND');
    }
  });

  it('createWebRtcTransport with unknown routerId throws NOT_FOUND', async () => {
    await expect(
      createWebRtcTransport({
        routerId: 'bogus-router',
        userId: 'user-3',
        direction: 'send',
      }),
    ).rejects.toMatchObject({ code: 'NOT_FOUND' });
  });
});
