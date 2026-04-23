import * as mediasoup from 'mediasoup';
import sensible from '@fastify/sensible';
import Fastify from 'fastify';

import { config } from './config.js';
import { registerErrorHandler } from './middlewares/error-handler.js';
import { internalAuthPlugin } from './middlewares/internal-auth.js';
import { closeWorker, getWorkerSnapshot, startWorker } from './mediasoup/worker.js';
import { consumerRoutes } from './routes/consumer.route.js';
import { producerRoutes } from './routes/producer.route.js';
import { routerRoutes } from './routes/router.route.js';
import { transportRoutes } from './routes/transport.route.js';
import { getRouterStats } from './services/router.service.js';
import { logger } from './utils/logger.js';

export async function buildApp() {
  const app = Fastify({
    loggerInstance: logger,
    disableRequestLogging: false,
    trustProxy: false,
    bodyLimit: 256 * 1024,
  });

  await app.register(sensible);
  registerErrorHandler(app);

  app.get('/healthz', async () => {
    const snapshot = getWorkerSnapshot();
    return {
      ok: snapshot.ready,
      service: 'media-server',
      mediasoupVersion: mediasoup.version,
      workerPid: snapshot.pid,
      workerRestartAttempts: snapshot.restartAttempts,
      uptimeSec: Math.round(process.uptime()),
      timestamp: new Date().toISOString(),
    };
  });

  app.get('/readyz', async (_request, reply) => {
    const snapshot = getWorkerSnapshot();
    if (!snapshot.ready) {
      reply.code(503);
      return {
        ready: false,
        reason: 'mediasoup worker not ready',
      };
    }
    return { ready: true };
  });

  await app.register(internalAuthPlugin);

  app.get('/internal/info', async () => {
    const snapshot = getWorkerSnapshot();
    const routerStats = getRouterStats();
    return {
      service: 'media-server',
      version: '0.1.0',
      mediasoupVersion: mediasoup.version,
      worker: snapshot,
      listen: {
        ip: config.mediasoup.listenIp,
        announcedIp: config.mediasoup.announcedIp ?? null,
        rtcMinPort: config.mediasoup.rtcMinPort,
        rtcMaxPort: config.mediasoup.rtcMaxPort,
      },
      stats: {
        routers: routerStats.total,
      },
      routers: routerStats.rooms,
    };
  });

  await app.register(
    async (scope) => {
      await scope.register(routerRoutes);
      await scope.register(transportRoutes);
      await scope.register(producerRoutes);
      await scope.register(consumerRoutes);
    },
    { prefix: '/internal/v1' },
  );

  return app;
}

async function bootstrap(): Promise<void> {
  const app = await buildApp();

  try {
    await startWorker();
  } catch (err) {
    logger.fatal(
      { err: err instanceof Error ? err.message : String(err) },
      'failed to start mediasoup worker, exiting',
    );
    process.exit(1);
  }

  try {
    await app.listen({ host: config.http.host, port: config.http.port });
    logger.info(
      {
        host: config.http.host,
        port: config.http.port,
        env: config.nodeEnv,
        mediasoup: {
          listenIp: config.mediasoup.listenIp,
          announcedIp: config.mediasoup.announcedIp ?? '(none)',
          rtcPortRange: `${config.mediasoup.rtcMinPort}-${config.mediasoup.rtcMaxPort}`,
        },
      },
      'media-server listening',
    );
  } catch (err) {
    logger.fatal(
      { err: err instanceof Error ? err.message : String(err) },
      'failed to start HTTP server',
    );
    process.exit(1);
  }

  const shutdown = async (signal: string): Promise<void> => {
    logger.info({ signal }, 'received shutdown signal');
    try {
      await app.close();
    } catch (err) {
      logger.warn(
        { err: err instanceof Error ? err.message : String(err) },
        'error while closing fastify',
      );
    }
    await closeWorker();
    process.exit(0);
  };

  process.on('SIGINT', () => void shutdown('SIGINT'));
  process.on('SIGTERM', () => void shutdown('SIGTERM'));
  process.on('unhandledRejection', (reason) => {
    logger.error({ reason }, 'unhandled promise rejection');
  });
  process.on('uncaughtException', (err) => {
    logger.fatal({ err: err.message, stack: err.stack }, 'uncaught exception');
    process.exit(1);
  });
}

const isEntryPoint =
  import.meta.url === `file://${process.argv[1]}` ||
  process.argv[1]?.endsWith('app.ts') ||
  process.argv[1]?.endsWith('app.js');

if (isEntryPoint) {
  void bootstrap();
}
