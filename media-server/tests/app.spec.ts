import type { FastifyInstance } from 'fastify';
import { afterAll, beforeAll, describe, expect, it } from 'vitest';

import { buildApp } from '../src/app.js';
import { config } from '../src/config.js';
import { closeWorker, startWorker } from '../src/mediasoup/worker.js';
import { _clearConsumerMap } from '../src/services/consumer.service.js';
import { _clearProducerMap } from '../src/services/producer.service.js';
import { _clearRouterMap } from '../src/services/router.service.js';
import { _clearTransportMap } from '../src/services/transport.service.js';

let app: FastifyInstance;
const token = config.internalToken;

beforeAll(async () => {
  app = (await buildApp()) as unknown as FastifyInstance;
  await app.ready();
  await startWorker();
});

afterAll(async () => {
  _clearConsumerMap();
  _clearProducerMap();
  _clearTransportMap();
  _clearRouterMap();
  await app.close();
  await closeWorker();
});

describe('HTTP layer: health + auth', () => {
  it('GET /healthz returns ok snapshot', async () => {
    const res = await app.inject({ method: 'GET', url: '/healthz' });
    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.service).toBe('media-server');
    expect(body.mediasoupVersion).toBeTruthy();
  });

  it('GET /readyz returns 200 when worker ready', async () => {
    const res = await app.inject({ method: 'GET', url: '/readyz' });
    expect(res.statusCode).toBe(200);
    expect(res.json().ready).toBe(true);
  });

  it('GET /internal/info without token returns 401', async () => {
    const res = await app.inject({ method: 'GET', url: '/internal/info' });
    expect(res.statusCode).toBe(401);
  });

  it('GET /internal/info with wrong token returns 401', async () => {
    const res = await app.inject({
      method: 'GET',
      url: '/internal/info',
      headers: { 'x-internal-token': 'wrong-token-wrong-token' },
    });
    expect(res.statusCode).toBe(401);
  });

  it('GET /internal/info with valid token returns worker info', async () => {
    const res = await app.inject({
      method: 'GET',
      url: '/internal/info',
      headers: { 'x-internal-token': token },
    });
    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.service).toBe('media-server');
    expect(body.mediasoupVersion).toBeTruthy();
  });
});

describe('HTTP layer: /internal/v1/* routes', () => {
  let routerId = '';
  let sendTransportId = '';
  let recvTransportId = '';

  it('POST /internal/v1/routers 201', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/internal/v1/routers',
      headers: { 'x-internal-token': token },
      payload: { roomCode: 'ROOM-HTTP-1' },
    });
    expect(res.statusCode).toBe(201);
    const body = res.json();
    expect(body.routerId).toBeTruthy();
    routerId = body.routerId;
  });

  it('POST /internal/v1/routers validation error returns 400', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/internal/v1/routers',
      headers: { 'x-internal-token': token },
      payload: { roomCode: 'lower-case' },
    });
    expect(res.statusCode).toBe(400);
    const body = res.json();
    expect(body.code).toBe('VALIDATION_ERROR');
    expect(Array.isArray(body.fieldErrors)).toBe(true);
  });

  it('POST /internal/v1/transports (send) 201', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/internal/v1/transports',
      headers: { 'x-internal-token': token },
      payload: { routerId, userId: 'u-1', direction: 'send' },
    });
    expect(res.statusCode).toBe(201);
    sendTransportId = res.json().id;
    expect(sendTransportId).toBeTruthy();
  });

  it('POST /internal/v1/transports (recv) 201', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/internal/v1/transports',
      headers: { 'x-internal-token': token },
      payload: { routerId, userId: 'u-2', direction: 'recv' },
    });
    expect(res.statusCode).toBe(201);
    recvTransportId = res.json().id;
    expect(recvTransportId).toBeTruthy();
  });

  it('POST /internal/v1/transports with unknown router returns 404', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/internal/v1/transports',
      headers: { 'x-internal-token': token },
      payload: { routerId: 'bogus-router', userId: 'u-3', direction: 'send' },
    });
    expect(res.statusCode).toBe(404);
    expect(res.json().code).toBe('NOT_FOUND');
  });

  const sampleAudioRtpParameters = {
    codecs: [
      { mimeType: 'audio/opus', clockRate: 48000, channels: 2, payloadType: 100 },
    ],
    encodings: [{ ssrc: 222222 }],
  };

  it('POST /internal/v1/producers with empty codecs returns 400 VALIDATION_ERROR', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/internal/v1/producers',
      headers: { 'x-internal-token': token },
      payload: {
        transportId: recvTransportId,
        kind: 'audio',
        rtpParameters: { codecs: [] },
      },
    });
    expect(res.statusCode).toBe(400);
    expect(res.json().code).toBe('VALIDATION_ERROR');
  });

  it('POST /internal/v1/producers with recv transport returns 409', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/internal/v1/producers',
      headers: { 'x-internal-token': token },
      payload: {
        transportId: recvTransportId,
        kind: 'audio',
        rtpParameters: sampleAudioRtpParameters,
      },
    });
    expect(res.statusCode).toBe(409);
    expect(res.json().code).toBe('CONFLICT');
  });

  it('POST /internal/v1/consumers with empty codecs returns 400 VALIDATION_ERROR', async () => {
    const producerRes = await app.inject({
      method: 'POST',
      url: '/internal/v1/producers',
      headers: { 'x-internal-token': token },
      payload: {
        transportId: sendTransportId,
        kind: 'audio',
        rtpParameters: sampleAudioRtpParameters,
      },
    });
    expect(producerRes.statusCode).toBe(201);
    const producerId = producerRes.json().id;

    const res = await app.inject({
      method: 'POST',
      url: '/internal/v1/consumers',
      headers: { 'x-internal-token': token },
      payload: {
        routerId,
        transportId: recvTransportId,
        producerId,
        rtpCapabilities: { codecs: [] },
      },
    });
    expect(res.statusCode).toBe(400);
    expect(res.json().code).toBe('VALIDATION_ERROR');
  });

  it('POST /internal/v1/consumers with incompatible codecs returns 400 CAN_NOT_CONSUME', async () => {
    const producerRes = await app.inject({
      method: 'POST',
      url: '/internal/v1/producers',
      headers: { 'x-internal-token': token },
      payload: {
        transportId: sendTransportId,
        kind: 'audio',
        rtpParameters: {
          ...sampleAudioRtpParameters,
          encodings: [{ ssrc: 333333 }],
        },
      },
    });
    expect(producerRes.statusCode).toBe(201);
    const producerId = producerRes.json().id;

    const res = await app.inject({
      method: 'POST',
      url: '/internal/v1/consumers',
      headers: { 'x-internal-token': token },
      payload: {
        routerId,
        transportId: recvTransportId,
        producerId,
        rtpCapabilities: {
          // 故意给一个 payloadType 不在 router 中的假 codec
          codecs: [{ mimeType: 'audio/unknown-fake', clockRate: 48000 }],
        },
      },
    });
    expect(res.statusCode).toBe(400);
    expect(res.json().code).toBe('CAN_NOT_CONSUME');
  });

  it('DELETE /internal/v1/producers/:id returns 404 for unknown', async () => {
    const res = await app.inject({
      method: 'DELETE',
      url: '/internal/v1/producers/unknown-id',
      headers: { 'x-internal-token': token },
    });
    expect(res.statusCode).toBe(404);
  });

  it('POST /internal/v1/consumers/:id/resume returns 404 for unknown', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/internal/v1/consumers/unknown-id/resume',
      headers: { 'x-internal-token': token },
    });
    expect(res.statusCode).toBe(404);
  });

  it('DELETE /internal/v1/consumers/:id returns 404 for unknown', async () => {
    const res = await app.inject({
      method: 'DELETE',
      url: '/internal/v1/consumers/unknown-id',
      headers: { 'x-internal-token': token },
    });
    expect(res.statusCode).toBe(404);
  });

  it('DELETE /internal/v1/routers/:routerId cleans up', async () => {
    const res = await app.inject({
      method: 'DELETE',
      url: `/internal/v1/routers/${routerId}`,
      headers: { 'x-internal-token': token },
    });
    expect(res.statusCode).toBe(200);
    expect(res.json().ok).toBe(true);
  });
});
