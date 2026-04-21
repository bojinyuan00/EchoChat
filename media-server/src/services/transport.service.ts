import type {
  DtlsParameters,
  IceCandidate,
  IceParameters,
  WebRtcTransport,
} from 'mediasoup/types';

import { config } from '../config.js';
import { AppError, conflict, notFound } from '../utils/errors.js';
import { childLogger } from '../utils/logger.js';
import { assertTestOnly } from '../utils/test-guard.js';

import { getRouter } from './router.service.js';
import type { TransportDirection } from '../schemas/transport.schema.js';

const log = childLogger({ module: 'services.transport' });

interface TransportEntry {
  transport: WebRtcTransport;
  routerId: string;
  userId: string;
  direction: TransportDirection;
  connected: boolean;
  createdAt: number;
}

const transportMap = new Map<string, TransportEntry>();

export interface CreatedTransportInfo {
  id: string;
  iceParameters: IceParameters;
  iceCandidates: IceCandidate[];
  dtlsParameters: DtlsParameters;
}

function buildListenIps(): Array<{ ip: string; announcedIp?: string }> {
  const entry: { ip: string; announcedIp?: string } = {
    ip: config.mediasoup.listenIp,
  };
  if (config.mediasoup.announcedIp) {
    entry.announcedIp = config.mediasoup.announcedIp;
  }
  return [entry];
}

export async function createWebRtcTransport(params: {
  routerId: string;
  userId: string;
  direction: TransportDirection;
}): Promise<CreatedTransportInfo> {
  const router = getRouter(params.routerId);

  const transport = await router.createWebRtcTransport({
    listenIps: buildListenIps(),
    enableUdp: true,
    enableTcp: true,
    preferUdp: true,
    initialAvailableOutgoingBitrate: 1_000_000,
    appData: {
      userId: params.userId,
      direction: params.direction,
      routerId: params.routerId,
    },
  });

  transport.observer.once('close', () => {
    transportMap.delete(transport.id);
    log.info(
      { transportId: transport.id, routerId: params.routerId, userId: params.userId },
      'transport closed and removed from map',
    );
  });

  transportMap.set(transport.id, {
    transport,
    routerId: params.routerId,
    userId: params.userId,
    direction: params.direction,
    connected: false,
    createdAt: Date.now(),
  });

  log.info(
    {
      transportId: transport.id,
      routerId: params.routerId,
      userId: params.userId,
      direction: params.direction,
    },
    'webrtc transport created',
  );

  return {
    id: transport.id,
    iceParameters: transport.iceParameters,
    iceCandidates: transport.iceCandidates,
    dtlsParameters: transport.dtlsParameters,
  };
}

export function getTransport(transportId: string): WebRtcTransport {
  const entry = transportMap.get(transportId);
  if (!entry) {
    throw notFound('transport', transportId);
  }
  return entry.transport;
}

export function getTransportEntry(transportId: string): TransportEntry {
  const entry = transportMap.get(transportId);
  if (!entry) {
    throw notFound('transport', transportId);
  }
  return entry;
}

export async function connectTransport(params: {
  transportId: string;
  dtlsParameters: DtlsParameters;
}): Promise<void> {
  const entry = transportMap.get(params.transportId);
  if (!entry) {
    throw notFound('transport', params.transportId);
  }
  // 乐观锁：先置位再 await，防止并发重复 connect 被 mediasoup 层包成 500
  if (entry.connected) {
    throw conflict(`transport already connected: ${params.transportId}`);
  }
  entry.connected = true;
  try {
    await entry.transport.connect({ dtlsParameters: params.dtlsParameters });
    log.info({ transportId: params.transportId }, 'transport connected');
  } catch (err) {
    entry.connected = false;
    const message = err instanceof Error ? err.message : String(err);
    throw new AppError('MEDIASOUP_ERROR', `failed to connect transport: ${message}`, {
      transportId: params.transportId,
    });
  }
}

export function getTransportStats(): { total: number } {
  return { total: transportMap.size };
}

/** 测试专用：复位 transportMap，生产环境调用会抛错 */
export function _clearTransportMap(): void {
  assertTestOnly('_clearTransportMap');
  for (const entry of transportMap.values()) {
    try {
      entry.transport.close();
    } catch {
      // already closed
    }
  }
  transportMap.clear();
}
