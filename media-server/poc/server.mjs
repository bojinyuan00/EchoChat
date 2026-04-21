// Phase 2e-2 Task 0: mediasoup PoC Server
// 架构：Fastify 提供静态资源 + WS 信令 + mediasoup 单 Worker 单 Router
// 仅用于验证技术栈，PoC 结束后可删除或演化为正式 media-server

import Fastify from 'fastify';
import fastifyStatic from '@fastify/static';
import fastifyWebsocket from '@fastify/websocket';
import * as mediasoup from 'mediasoup';
import pino from 'pino';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomUUID } from 'node:crypto';
import os from 'node:os';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const logger = pino({
  transport: {
    target: 'pino-pretty',
    options: { colorize: true, translateTime: 'SYS:HH:MM:ss.l' },
  },
});

// ---------------- Config ----------------

const HTTP_PORT = Number(process.env.HTTP_PORT ?? 3300);
const LISTEN_IP = process.env.MEDIASOUP_LISTEN_IP ?? '0.0.0.0';
const ANNOUNCED_IP = process.env.MEDIASOUP_ANNOUNCED_IP ?? ''; // 本机留空走 iceCandidates 自动
const RTC_MIN_PORT = Number(process.env.MEDIASOUP_RTC_MIN_PORT ?? 40000);
const RTC_MAX_PORT = Number(process.env.MEDIASOUP_RTC_MAX_PORT ?? 40099);

// 媒体编解码（Router 创建时注入）
const mediaCodecs = [
  {
    kind: 'audio',
    mimeType: 'audio/opus',
    clockRate: 48000,
    channels: 2,
  },
  {
    kind: 'video',
    mimeType: 'video/VP8',
    clockRate: 90000,
    parameters: { 'x-google-start-bitrate': 1000 },
  },
  {
    kind: 'video',
    mimeType: 'video/H264',
    clockRate: 90000,
    parameters: {
      'packetization-mode': 1,
      'profile-level-id': '42e01f',
      'level-asymmetry-allowed': 1,
    },
  },
];

// ---------------- Mediasoup 全局资源 ----------------

let worker;
let router;

// peers：peerId → { ws, transports: Map, producers: Map, consumers: Map }
const peers = new Map();

async function initMediasoup() {
  worker = await mediasoup.createWorker({
    rtcMinPort: RTC_MIN_PORT,
    rtcMaxPort: RTC_MAX_PORT,
    logLevel: 'warn',
  });

  worker.on('died', (err) => {
    logger.error({ err }, 'mediasoup worker died, exiting');
    process.exit(1);
  });

  router = await worker.createRouter({ mediaCodecs });

  logger.info(
    {
      workerPid: worker.pid,
      routerId: router.id,
      rtcPortRange: `${RTC_MIN_PORT}-${RTC_MAX_PORT}`,
      listenIp: LISTEN_IP,
      announcedIp: ANNOUNCED_IP || '(unset, use local LAN ip)',
    },
    'mediasoup worker + router ready',
  );
}

// ---------------- Transport 创建辅助 ----------------

async function createWebRtcTransport() {
  const listenIps = [{ ip: LISTEN_IP, announcedIp: ANNOUNCED_IP || undefined }];
  const transport = await router.createWebRtcTransport({
    listenIps,
    enableUdp: true,
    enableTcp: true,
    preferUdp: true,
    initialAvailableOutgoingBitrate: 1_000_000,
  });
  return transport;
}

// ---------------- WS 信令 ----------------

function send(ws, type, data = {}, reqId) {
  if (ws.readyState !== 1) return;
  const msg = { type, data };
  if (reqId) msg.reqId = reqId;
  ws.send(JSON.stringify(msg));
}

function broadcast(excludeId, type, data) {
  for (const [pid, peer] of peers) {
    if (pid === excludeId) continue;
    send(peer.ws, type, data);
  }
}

async function handleMessage(peerId, peer, raw) {
  let msg;
  try {
    msg = JSON.parse(raw);
  } catch {
    logger.warn({ peerId, raw: String(raw).slice(0, 80) }, 'invalid json');
    return;
  }
  const { type, data = {}, reqId } = msg;

  try {
    switch (type) {
      case 'getRtpCapabilities': {
        send(peer.ws, 'rtpCapabilities', router.rtpCapabilities, reqId);
        break;
      }
      case 'createTransport': {
        const transport = await createWebRtcTransport();
        peer.transports.set(transport.id, transport);
        send(
          peer.ws,
          'transportCreated',
          {
            direction: data.direction,
            id: transport.id,
            iceParameters: transport.iceParameters,
            iceCandidates: transport.iceCandidates,
            dtlsParameters: transport.dtlsParameters,
          },
          reqId,
        );
        break;
      }
      case 'connectTransport': {
        const { transportId, dtlsParameters } = data;
        const transport = peer.transports.get(transportId);
        if (!transport) throw new Error(`transport ${transportId} not found`);
        await transport.connect({ dtlsParameters });
        send(peer.ws, 'transportConnected', { transportId }, reqId);
        break;
      }
      case 'produce': {
        const { transportId, kind, rtpParameters } = data;
        const transport = peer.transports.get(transportId);
        if (!transport) throw new Error(`transport ${transportId} not found`);
        const producer = await transport.produce({ kind, rtpParameters });
        peer.producers.set(producer.id, producer);

        producer.on('transportclose', () => {
          peer.producers.delete(producer.id);
        });

        send(peer.ws, 'produced', { producerId: producer.id, kind }, reqId);

        // 广播给其他 peer
        broadcast(peerId, 'newProducer', {
          peerId,
          producerId: producer.id,
          kind,
        });
        logger.info({ peerId, producerId: producer.id, kind }, 'peer produced');
        break;
      }
      case 'consume': {
        const { transportId, producerId, rtpCapabilities } = data;
        const transport = peer.transports.get(transportId);
        if (!transport) throw new Error(`transport ${transportId} not found`);
        if (!router.canConsume({ producerId, rtpCapabilities })) {
          throw new Error(`router cannot consume producer ${producerId}`);
        }
        const consumer = await transport.consume({
          producerId,
          rtpCapabilities,
          paused: true, // 先暂停，客户端确认后再 resume
        });
        peer.consumers.set(consumer.id, consumer);

        consumer.on('transportclose', () => {
          peer.consumers.delete(consumer.id);
        });
        consumer.on('producerclose', () => {
          peer.consumers.delete(consumer.id);
          send(peer.ws, 'consumerClosed', { consumerId: consumer.id });
        });

        send(
          peer.ws,
          'consumed',
          {
            id: consumer.id,
            producerId,
            kind: consumer.kind,
            rtpParameters: consumer.rtpParameters,
          },
          reqId,
        );
        break;
      }
      case 'resumeConsumer': {
        const { consumerId } = data;
        const consumer = peer.consumers.get(consumerId);
        if (!consumer) throw new Error(`consumer ${consumerId} not found`);
        await consumer.resume();
        send(peer.ws, 'consumerResumed', { consumerId }, reqId);
        break;
      }
      case 'getPeers': {
        // 返回当前其他 peers 正在 produce 的 producerId 列表
        const others = [];
        for (const [pid, p] of peers) {
          if (pid === peerId) continue;
          for (const producer of p.producers.values()) {
            others.push({ peerId: pid, producerId: producer.id, kind: producer.kind });
          }
        }
        send(peer.ws, 'peers', { items: others }, reqId);
        break;
      }
      default:
        send(peer.ws, 'error', { message: `unknown type: ${type}` }, reqId);
    }
  } catch (err) {
    logger.error({ peerId, type, err: err.message }, 'handle message failed');
    send(peer.ws, 'error', { message: err.message }, reqId);
  }
}

function cleanupPeer(peerId) {
  const peer = peers.get(peerId);
  if (!peer) return;
  for (const p of peer.producers.values()) p.close();
  for (const c of peer.consumers.values()) c.close();
  for (const t of peer.transports.values()) t.close();
  peers.delete(peerId);
  broadcast(peerId, 'peerLeft', { peerId });
  logger.info(
    {
      peerId,
      remaining: peers.size,
      routerStats: { producers: [...peers.values()].reduce((n, p) => n + p.producers.size, 0) },
    },
    'peer cleaned up',
  );
}

// ---------------- Fastify 启动 ----------------

async function main() {
  await initMediasoup();

  const app = Fastify({ logger: false });

  await app.register(fastifyWebsocket);
  await app.register(fastifyStatic, {
    root: path.join(__dirname, 'public'),
    prefix: '/',
  });

  app.get('/healthz', async () => ({
    ok: true,
    workerPid: worker.pid,
    routerId: router.id,
    peers: peers.size,
  }));

  app.get('/stats', async () => {
    // 基础压测观测：peer 数 + producer/consumer 总数 + 内存
    let totalProducers = 0;
    let totalConsumers = 0;
    let totalTransports = 0;
    for (const p of peers.values()) {
      totalProducers += p.producers.size;
      totalConsumers += p.consumers.size;
      totalTransports += p.transports.size;
    }
    const mem = process.memoryUsage();
    return {
      peers: peers.size,
      transports: totalTransports,
      producers: totalProducers,
      consumers: totalConsumers,
      memoryMB: {
        rss: Math.round(mem.rss / 1024 / 1024),
        heapUsed: Math.round(mem.heapUsed / 1024 / 1024),
      },
      loadavg: os.loadavg(),
      uptime: Math.round(process.uptime()),
    };
  });

  app.register(async function (f) {
    f.get('/ws', { websocket: true }, (socket, req) => {
      const peerId = randomUUID();
      const peer = {
        ws: socket,
        transports: new Map(),
        producers: new Map(),
        consumers: new Map(),
      };
      peers.set(peerId, peer);

      logger.info({ peerId, total: peers.size, ua: req.headers['user-agent'] }, 'peer connected');
      send(socket, 'welcome', { peerId });

      socket.on('message', (raw) => {
        handleMessage(peerId, peer, raw.toString());
      });

      socket.on('close', () => {
        logger.info({ peerId }, 'ws closed');
        cleanupPeer(peerId);
      });

      socket.on('error', (err) => {
        logger.warn({ peerId, err: err.message }, 'ws error');
      });
    });
  });

  await app.listen({ port: HTTP_PORT, host: '0.0.0.0' });
  logger.info(`PoC ready: http://localhost:${HTTP_PORT}  (open in 2 browser windows)`);
}

main().catch((err) => {
  logger.error({ err }, 'poc server failed to start');
  process.exit(1);
});
