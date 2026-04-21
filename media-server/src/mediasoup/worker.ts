import * as mediasoup from 'mediasoup';
import type { Worker, WorkerLogLevel } from 'mediasoup/types';

import { config } from '../config.js';
import { childLogger } from '../utils/logger.js';

const log = childLogger({ module: 'mediasoup.worker' });

interface WorkerState {
  worker: Worker | null;
  starting: Promise<Worker> | null;
  restartTimer: NodeJS.Timeout | null;
  restartAttempts: number;
}

const state: WorkerState = {
  worker: null,
  starting: null,
  restartTimer: null,
  restartAttempts: 0,
};

const MAX_RESTART_BACKOFF_MS = 30_000;

async function spawnWorker(): Promise<Worker> {
  const worker = await mediasoup.createWorker({
    logLevel: config.mediasoup.workerLogLevel as WorkerLogLevel,
    rtcMinPort: config.mediasoup.rtcMinPort,
    rtcMaxPort: config.mediasoup.rtcMaxPort,
  });

  log.info(
    {
      pid: worker.pid,
      rtcMinPort: config.mediasoup.rtcMinPort,
      rtcMaxPort: config.mediasoup.rtcMaxPort,
    },
    'mediasoup worker started',
  );

  worker.on('died', (error) => {
    log.error(
      { pid: worker.pid, err: error instanceof Error ? error.message : String(error) },
      'mediasoup worker died, scheduling restart',
    );
    state.worker = null;
    scheduleRestart();
  });

  return worker;
}

function scheduleRestart(): void {
  if (state.restartTimer) {
    return;
  }
  state.restartAttempts += 1;
  const delay = Math.min(1_000 * 2 ** (state.restartAttempts - 1), MAX_RESTART_BACKOFF_MS);

  log.warn({ attempt: state.restartAttempts, delayMs: delay }, 'scheduling worker restart');

  state.restartTimer = setTimeout(() => {
    state.restartTimer = null;
    startWorker().catch((err) => {
      log.error(
        { err: err instanceof Error ? err.message : String(err) },
        'worker restart failed, will retry',
      );
      scheduleRestart();
    });
  }, delay);
}

export async function startWorker(): Promise<Worker> {
  if (state.worker) {
    return state.worker;
  }
  if (state.starting) {
    return state.starting;
  }
  state.starting = spawnWorker()
    .then((worker) => {
      state.worker = worker;
      state.starting = null;
      state.restartAttempts = 0;
      return worker;
    })
    .catch((err) => {
      state.starting = null;
      throw err;
    });
  return state.starting;
}

export function getWorker(): Worker {
  if (!state.worker) {
    throw new Error('mediasoup worker is not ready');
  }
  return state.worker;
}

export function getWorkerSnapshot(): {
  ready: boolean;
  pid: number | null;
  restartAttempts: number;
} {
  return {
    ready: state.worker !== null,
    pid: state.worker?.pid ?? null,
    restartAttempts: state.restartAttempts,
  };
}

export async function closeWorker(): Promise<void> {
  if (state.restartTimer) {
    clearTimeout(state.restartTimer);
    state.restartTimer = null;
  }
  if (state.worker) {
    const w = state.worker;
    state.worker = null;
    try {
      w.close();
      log.info({ pid: w.pid }, 'mediasoup worker closed');
    } catch (err) {
      log.warn(
        { err: err instanceof Error ? err.message : String(err) },
        'error while closing worker',
      );
    }
  }
}
