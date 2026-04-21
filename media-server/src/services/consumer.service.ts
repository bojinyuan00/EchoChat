import type { Consumer, MediaKind, RtpCapabilities, RtpParameters } from 'mediasoup/types';

import { AppError, notFound } from '../utils/errors.js';
import { childLogger } from '../utils/logger.js';
import { assertTestOnly } from '../utils/test-guard.js';

// 注：getProducer 仍然保留，既用于 notFound 语义（消费不存在的 producer 直接返回 404），
// 也用于后续 Phase 2e-3 需要预读 producer.appData 做资格校验
import { getProducer } from './producer.service.js';
import { getRouter } from './router.service.js';
import { getTransportEntry } from './transport.service.js';

const log = childLogger({ module: 'services.consumer' });

interface ConsumerEntry {
  consumer: Consumer;
  transportId: string;
  producerId: string;
  routerId: string;
  userId: string;
  createdAt: number;
}

const consumerMap = new Map<string, ConsumerEntry>();

export interface CreatedConsumerInfo {
  id: string;
  kind: MediaKind;
  rtpParameters: RtpParameters;
  producerPaused: boolean;
}

export async function createConsumer(params: {
  routerId: string;
  transportId: string;
  producerId: string;
  rtpCapabilities: RtpCapabilities;
}): Promise<CreatedConsumerInfo> {
  const router = getRouter(params.routerId);
  // 预检 producer 存在性；后续不再读 producer.paused，改用 consumer.producerPaused 更符合 mediasoup 语义
  getProducer(params.producerId);
  const transportEntry = getTransportEntry(params.transportId);

  if (transportEntry.direction !== 'recv') {
    throw new AppError(
      'CONFLICT',
      `transport ${params.transportId} is not a recv transport (direction=${transportEntry.direction})`,
      { transportId: params.transportId, direction: transportEntry.direction },
    );
  }

  if (!router.canConsume({ producerId: params.producerId, rtpCapabilities: params.rtpCapabilities })) {
    throw new AppError(
      'CAN_NOT_CONSUME',
      `router cannot consume producer ${params.producerId} with given rtpCapabilities`,
      { routerId: params.routerId, producerId: params.producerId },
    );
  }

  try {
    const consumer = await transportEntry.transport.consume({
      producerId: params.producerId,
      rtpCapabilities: params.rtpCapabilities,
      paused: true,
    });

    consumer.observer.once('close', () => {
      consumerMap.delete(consumer.id);
      log.info(
        { consumerId: consumer.id, transportId: params.transportId },
        'consumer closed and removed from map',
      );
    });

    consumer.once('producerclose', () => {
      log.info({ consumerId: consumer.id }, 'upstream producer closed, closing consumer');
      consumer.close();
    });

    consumerMap.set(consumer.id, {
      consumer,
      transportId: params.transportId,
      producerId: params.producerId,
      routerId: params.routerId,
      userId: transportEntry.userId,
      createdAt: Date.now(),
    });

    log.info(
      {
        consumerId: consumer.id,
        transportId: params.transportId,
        producerId: params.producerId,
        kind: consumer.kind,
      },
      'consumer created (paused)',
    );

    return {
      id: consumer.id,
      kind: consumer.kind,
      rtpParameters: consumer.rtpParameters,
      producerPaused: consumer.producerPaused,
    };
  } catch (err) {
    if (err instanceof AppError) {
      throw err;
    }
    const message = err instanceof Error ? err.message : String(err);
    throw new AppError('MEDIASOUP_ERROR', `failed to create consumer: ${message}`, {
      transportId: params.transportId,
      producerId: params.producerId,
    });
  }
}

export async function resumeConsumer(consumerId: string): Promise<void> {
  const entry = consumerMap.get(consumerId);
  if (!entry) {
    throw notFound('consumer', consumerId);
  }
  await entry.consumer.resume();
  log.info({ consumerId }, 'consumer resumed');
}

export async function closeConsumer(consumerId: string): Promise<void> {
  const entry = consumerMap.get(consumerId);
  if (!entry) {
    throw notFound('consumer', consumerId);
  }
  entry.consumer.close();
  log.info({ consumerId }, 'consumer closed explicitly');
}

export function getConsumerStats(): { total: number } {
  return { total: consumerMap.size };
}

/** 测试专用：复位 consumerMap，生产环境调用会抛错 */
export function _clearConsumerMap(): void {
  assertTestOnly('_clearConsumerMap');
  for (const entry of consumerMap.values()) {
    try {
      entry.consumer.close();
    } catch {
      // already closed
    }
  }
  consumerMap.clear();
}
