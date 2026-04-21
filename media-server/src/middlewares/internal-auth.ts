import { timingSafeEqual } from 'node:crypto';

import type { FastifyReply, FastifyRequest } from 'fastify';
import fp from 'fastify-plugin';

import { config } from '../config.js';

const INTERNAL_TOKEN_HEADER = 'x-internal-token';

/**
 * 反向白名单：只有命中 PRIVATE_PATH_PREFIXES 的请求才需要校验 internal token。
 * 其余路径（/healthz / /readyz / 未来的 /metrics / /docs 等）默认开放，
 * 避免新增公共端点时漏加白名单导致 CI/监控被 401。
 */
const PRIVATE_PATH_PREFIXES = ['/internal/'];

function isPrivatePath(url: string): boolean {
  return PRIVATE_PATH_PREFIXES.some((prefix) => url === prefix.slice(0, -1) || url.startsWith(prefix));
}

function safeEqual(a: string, b: string): boolean {
  const aBuf = Buffer.from(a, 'utf-8');
  const bBuf = Buffer.from(b, 'utf-8');
  if (aBuf.length !== bBuf.length) {
    return false;
  }
  return timingSafeEqual(aBuf, bBuf);
}

async function internalAuthHook(request: FastifyRequest, reply: FastifyReply): Promise<void> {
  if (!isPrivatePath(request.url)) {
    return;
  }

  const rawHeader = request.headers[INTERNAL_TOKEN_HEADER];
  const token = Array.isArray(rawHeader) ? rawHeader[0] : rawHeader;

  if (!token || !safeEqual(token, config.internalToken)) {
    request.log.warn(
      { path: request.url, method: request.method, hasToken: Boolean(token) },
      'internal auth rejected',
    );
    reply.code(401).send({
      code: 'UNAUTHORIZED',
      message: 'missing or invalid X-Internal-Token',
    });
  }
}

export const internalAuthPlugin = fp(
  async (fastify) => {
    fastify.addHook('onRequest', internalAuthHook);
  },
  {
    name: 'internal-auth',
  },
);
