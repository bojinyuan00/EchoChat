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

/**
 * 提取 URL 的 pathname，去除 query string / hash。
 *
 * Task 16 Nit（代码审查 2026-04-23）：之前直接用 `request.url` 做 startsWith 匹配，
 * 对 `/healthz?x=/internal/...` 这种带 query 的请求是稳定的（因为 query 在路径之后），
 * 但对构造如 `/internal/foo` 开头但含 `?` 的路径日志会出现误判；
 * 同时更严格的语义要求按 route path 决策而非 raw URL。
 *
 * onRequest 阶段（fastify 路由匹配之前）`request.routerPath` 尚未赋值，
 * 因此这里手动按 `?` / `#` 截断再与白名单比较，保证只依赖 path 本身。
 */
function extractPath(rawUrl: string): string {
  const qIdx = rawUrl.indexOf('?');
  const hIdx = rawUrl.indexOf('#');
  let end = rawUrl.length;
  if (qIdx !== -1) end = Math.min(end, qIdx);
  if (hIdx !== -1) end = Math.min(end, hIdx);
  return rawUrl.slice(0, end);
}

function isPrivatePath(rawUrl: string): boolean {
  const path = extractPath(rawUrl);
  return PRIVATE_PATH_PREFIXES.some((prefix) => path === prefix.slice(0, -1) || path.startsWith(prefix));
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
