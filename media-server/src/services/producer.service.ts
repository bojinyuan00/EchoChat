import type { MediaKind, Producer, RtpParameters } from 'mediasoup/types';

import { AppError, notFound } from '../utils/errors.js';
import { childLogger } from '../utils/logger.js';
import { assertTestOnly } from '../utils/test-guard.js';

import { getTransportEntry } from './transport.service.js';

const log = childLogger({ module: 'services.producer' });

interface ProducerEntry {
  producer: Producer;
  transportId: string;
  userId: string;
  createdAt: number;
}

const producerMap = new Map<string, ProducerEntry>();

export async function createProducer(params: {
  transportId: string;
  kind: MediaKind;
  rtpParameters: RtpParameters;
  appData?: Record<string, unknown>;
}): Promise<{ id: string }> {
  const entry = getTransportEntry(params.transportId);
  if (entry.direction !== 'send') {
    throw new AppError(
      'CONFLICT',
      `transport ${params.transportId} is not a send transport (direction=${entry.direction})`,
      { transportId: params.transportId, direction: entry.direction },
    );
  }

  try {
    const producer = await entry.transport.produce({
      kind: params.kind,
      rtpParameters: params.rtpParameters,
      appData: {
        ...(params.appData ?? {}),
        userId: entry.userId,
      },
    });

    producer.observer.once('close', () => {
      producerMap.delete(producer.id);
      log.info(
        { producerId: producer.id, transportId: params.transportId },
        'producer closed and removed from map',
      );
    });

    producerMap.set(producer.id, {
      producer,
      transportId: params.transportId,
      userId: entry.userId,
      createdAt: Date.now(),
    });

    log.info(
      {
        producerId: producer.id,
        transportId: params.transportId,
        userId: entry.userId,
        kind: params.kind,
      },
      'producer created',
    );

    return { id: producer.id };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    throw new AppError('MEDIASOUP_ERROR', `failed to create producer: ${message}`, {
      transportId: params.transportId,
      kind: params.kind,
    });
  }
}

export function getProducer(producerId: string): Producer {
  const entry = producerMap.get(producerId);
  if (!entry) {
    throw notFound('producer', producerId);
  }
  return entry.producer;
}

export async function closeProducer(producerId: string): Promise<void> {
  const entry = producerMap.get(producerId);
  if (!entry) {
    throw notFound('producer', producerId);
  }
  entry.producer.close();
  log.info({ producerId }, 'producer closed explicitly');
}

export function getProducerStats(): { total: number } {
  return { total: producerMap.size };
}

/** 测试专用：复位 producerMap，生产环境调用会抛错 */
export function _clearProducerMap(): void {
  assertTestOnly('_clearProducerMap');
  for (const entry of producerMap.values()) {
    try {
      entry.producer.close();
    } catch {
      // already closed
    }
  }
  producerMap.clear();
}
