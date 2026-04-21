import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest';

import { config } from '../../src/config.js';
import { closeWorker, startWorker } from '../../src/mediasoup/worker.js';
import {
  _clearRouterMap,
  closeRouter,
  createRouter,
  getRouter,
  getRouterStats,
  tryGetRouter,
} from '../../src/services/router.service.js';
import { AppError } from '../../src/utils/errors.js';

beforeAll(async () => {
  await startWorker();
});

afterAll(async () => {
  _clearRouterMap();
  await closeWorker();
});

afterEach(() => {
  _clearRouterMap();
});

describe('router.service', () => {
  it('creates a router and returns rtpCapabilities', async () => {
    const { routerId, rtpCapabilities } = await createRouter('ROOM-TEST-1');
    expect(routerId).toBeTruthy();
    expect(Array.isArray(rtpCapabilities.codecs)).toBe(true);
    expect(rtpCapabilities.codecs?.length).toBeGreaterThan(0);

    const stats = getRouterStats();
    expect(stats.total).toBe(1);
    expect(stats.rooms[0]?.roomCode).toBe('ROOM-TEST-1');
  });

  it('getRouter throws NOT_FOUND for unknown id', () => {
    try {
      getRouter('non-existent-id');
      throw new Error('expected throw');
    } catch (err) {
      expect(err).toBeInstanceOf(AppError);
      expect((err as AppError).code).toBe('NOT_FOUND');
    }
  });

  it('tryGetRouter returns undefined instead of throwing', () => {
    expect(tryGetRouter('non-existent-id')).toBeUndefined();
  });

  it('closeRouter removes entry via observer close event', async () => {
    const { routerId } = await createRouter('ROOM-TEST-2');
    expect(getRouterStats().total).toBe(1);

    await closeRouter(routerId);
    // observer.once('close') executes synchronously when router.close() is called
    expect(getRouterStats().total).toBe(0);
  });

  it('closeRouter throws NOT_FOUND for unknown router', async () => {
    await expect(closeRouter('non-existent-id')).rejects.toBeInstanceOf(AppError);
  });

  it('enforces router limit', async () => {
    const originalLimit = config.mediasoup.maxRouters;
    (config.mediasoup as { maxRouters: number }).maxRouters = 2;

    try {
      await createRouter('ROOM-LIMIT-1');
      await createRouter('ROOM-LIMIT-2');
      await expect(createRouter('ROOM-LIMIT-3')).rejects.toMatchObject({
        code: 'ROUTER_LIMIT_EXCEEDED',
      });
    } finally {
      (config.mediasoup as { maxRouters: number }).maxRouters = originalLimit;
    }
  });
});
