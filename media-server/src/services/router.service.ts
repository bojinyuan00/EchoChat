import type { Router, RtpCapabilities } from 'mediasoup/types';

import { config } from '../config.js';
import { MEDIA_CODECS } from '../mediasoup/codecs.js';
import { getWorker } from '../mediasoup/worker.js';
import { AppError, notFound } from '../utils/errors.js';
import { childLogger } from '../utils/logger.js';
import { assertTestOnly } from '../utils/test-guard.js';

const log = childLogger({ module: 'services.router' });

interface RouterEntry {
  router: Router;
  roomCode: string;
  createdAt: number;
}

const routerMap = new Map<string, RouterEntry>();

export async function createRouter(roomCode: string): Promise<{
  routerId: string;
  rtpCapabilities: RtpCapabilities;
}> {
  if (routerMap.size >= config.mediasoup.maxRouters) {
    throw new AppError(
      'ROUTER_LIMIT_EXCEEDED',
      `router limit reached: ${config.mediasoup.maxRouters}`,
      { current: routerMap.size, max: config.mediasoup.maxRouters },
    );
  }

  const worker = getWorker();
  const router = await worker.createRouter({ mediaCodecs: MEDIA_CODECS });

  router.observer.once('close', () => {
    routerMap.delete(router.id);
    log.info({ routerId: router.id, roomCode }, 'router closed and removed from map');
  });

  routerMap.set(router.id, {
    router,
    roomCode,
    createdAt: Date.now(),
  });

  log.info({ routerId: router.id, roomCode, total: routerMap.size }, 'router created');
  return {
    routerId: router.id,
    rtpCapabilities: router.rtpCapabilities,
  };
}

export function getRouter(routerId: string): Router {
  const entry = routerMap.get(routerId);
  if (!entry) {
    throw notFound('router', routerId);
  }
  return entry.router;
}

export function tryGetRouter(routerId: string): Router | undefined {
  return routerMap.get(routerId)?.router;
}

export async function closeRouter(routerId: string): Promise<void> {
  const entry = routerMap.get(routerId);
  if (!entry) {
    throw notFound('router', routerId);
  }
  entry.router.close();
  log.info({ routerId, roomCode: entry.roomCode }, 'router closed explicitly');
}

export function getRouterStats(): {
  total: number;
  rooms: Array<{ routerId: string; roomCode: string; ageMs: number }>;
} {
  const now = Date.now();
  return {
    total: routerMap.size,
    rooms: [...routerMap.entries()].map(([routerId, entry]) => ({
      routerId,
      roomCode: entry.roomCode,
      ageMs: now - entry.createdAt,
    })),
  };
}

/** 测试专用：复位 routerMap，生产环境调用会抛错 */
export function _clearRouterMap(): void {
  assertTestOnly('_clearRouterMap');
  for (const entry of routerMap.values()) {
    try {
      entry.router.close();
    } catch {
      // already closed
    }
  }
  routerMap.clear();
}
