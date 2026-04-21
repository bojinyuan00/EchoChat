#!/usr/bin/env node
// Task 7 端到端验证脚本
// 目标：验证 Go HTTPMediaOrchestrator 真正调通 Node media-server
// 覆盖：Router 创建 / Transport 创建 / Producer 幂等关闭 404 映射 / Router 销毁

import WebSocket from 'ws';

const GO = 'http://localhost:8085/api/v1';
const NODE = 'http://localhost:3300';
const NODE_TOKEN = 'dev-token-abcdef1234567890';

let pass = 0;
let fail = 0;
const assert = (cond, name, detail) => {
  if (cond) {
    pass++;
    console.log(`PASS: ${name}`);
  } else {
    fail++;
    console.log(`FAIL: ${name}`, detail ?? '');
  }
};

// -------- HTTP utils --------
async function httpJSON(method, url, body, headers = {}) {
  const opts = { method, headers: { 'Content-Type': 'application/json', ...headers } };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const r = await fetch(url, opts);
  const text = await r.text();
  let data = null;
  try { data = JSON.parse(text); } catch (_) { /* non-json */ }
  return { status: r.status, data, text };
}

async function register(suffix) {
  const email = `t7user${suffix}@test.local`;
  const password = 'TestPassword123!';
  const username = `t7user${suffix}`;
  await httpJSON('POST', `${GO}/auth/register`, { email, username, password });
  const login = await httpJSON('POST', `${GO}/auth/login`, { account: email, password });
  if (!login.data?.data?.token) throw new Error(`login failed ${JSON.stringify(login)}`);
  return login.data.data.token;
}

// -------- WS utils --------
function connectWS(token) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(`ws://localhost:8085/ws?token=${token}`);
    ws.on('open', () => resolve(ws));
    ws.on('error', reject);
    setTimeout(() => reject(new Error('ws open timeout')), 5000);
  });
}

let _seqCounter = 1;
function wsSend(ws, event, data) {
  return new Promise((resolve) => {
    const seq = _seqCounter++;
    const ackEvent = `${event}.ack`;
    const onMsg = (raw) => {
      try {
        const m = JSON.parse(raw.toString());
        if (m.event === ackEvent && m.seq === seq) {
          ws.off('message', onMsg);
          resolve(m);
        }
      } catch {}
    };
    ws.on('message', onMsg);
    ws.send(JSON.stringify({ event, seq, data, time: new Date().toISOString() }));
    setTimeout(() => {
      ws.off('message', onMsg);
      resolve(null);
    }, 6000);
  });
}

// -------- Main --------
(async () => {
  console.log('==== Task 7 HTTPMediaOrchestrator E2E Verify ====');

  // 预检：media-server 直连 (必须通)
  const mHealth = await httpJSON('GET', `${NODE}/healthz`);
  assert(mHealth.status === 200 && mHealth.data?.ok === true, 'media-server healthz');
  const mStats0 = await httpJSON('GET', `${NODE}/internal/info`, undefined, { 'X-Internal-Token': NODE_TOKEN });
  assert(mStats0.status === 200, 'media-server /internal/info (token ok)');

  // 验证 Go→Node 鉴权闭合（错 token 应被拒）
  const badToken = await httpJSON('GET', `${NODE}/internal/info`, undefined, { 'X-Internal-Token': 'wrong' });
  assert(badToken.status === 401, 'media-server rejects wrong token (401)');

  // Step 1: 注册 2 个用户
  const ts = Date.now().toString().slice(-6);
  const tokenA = await register(`a${ts}`);
  const tokenB = await register(`b${ts}`);
  assert(!!tokenA && !!tokenB, 'register + login 2 users');

  // Step 2: 主持人 A 创建会议 —— 应在 Node 侧真实创建 Router
  const createRes = await httpJSON('POST', `${GO}/meeting/rooms`, {
    title: 'Task 7 Verify Room',
    type: 1,
  }, { Authorization: `Bearer ${tokenA}` });
  assert(createRes.status === 201, 'POST /meeting/rooms returns 201', createRes.data);
  const roomCode = createRes.data?.data?.room?.room_code;
  const roomID = createRes.data?.data?.room?.id;
  assert(typeof roomCode === 'string' && roomCode.length > 0, 'room_code returned', createRes.data);

  // Step 3: 用户 B 加入会议 (REST /join)
  const joinRes = await httpJSON('POST', `${GO}/meeting/rooms/${roomCode}/join`, {}, {
    Authorization: `Bearer ${tokenB}`,
  });
  assert(joinRes.status === 200, 'user B join success');

  // Step 4: A 与 B 建立 WS 连接
  const wsA = await connectWS(tokenA);
  const wsB = await connectWS(tokenB);

  // Step 5: WS room.join （以 WS 维度进入会议）
  const joinWSA = await wsSend(wsA, 'meeting.room.join', { room_code: roomCode });
  assert(joinWSA?.code === 0, 'A WS room.join ok', joinWSA);
  const joinWSB = await wsSend(wsB, 'meeting.room.join', { room_code: roomCode });
  assert(joinWSB?.code === 0, 'B WS room.join ok', joinWSB);

  // Step 6: A transport.create (send)
  const txCreateRes = await wsSend(wsA, 'meeting.transport.create', {
    room_code: roomCode,
    direction: 'send',
  });
  assert(txCreateRes?.code === 0, 'A transport.create returned ok', txCreateRes);
  const tx = txCreateRes?.data;
  assert(
    tx && typeof tx.id === 'string' && tx.id.length > 0 && !tx.id.startsWith('noop-'),
    'transport.id is real (not noop-)',
    tx,
  );
  assert(
    tx?.iceParameters && typeof tx.iceParameters === 'object',
    'iceParameters is an object (non-empty from Node)',
    tx,
  );
  assert(
    Array.isArray(tx?.iceCandidates) && tx.iceCandidates.length > 0,
    'iceCandidates[] non-empty (proves real mediasoup transport)',
    tx,
  );
  assert(
    tx?.dtlsParameters?.fingerprints && Array.isArray(tx.dtlsParameters.fingerprints),
    'dtlsParameters.fingerprints[] present',
    tx,
  );

  // Step 7: 幂等关闭不存在的 producer —— 404 映射测试
  const fakeProdClose = await wsSend(wsA, 'meeting.producer.close', {
    room_code: roomCode,
    producer_id: 'nonexistent-producer-id-9999',
  });
  assert(fakeProdClose?.code === 0, '404 mapped to ok (idempotent close)', fakeProdClose);

  // Step 8: 结束会议 —— 触发 CloseRouter
  const endRes = await httpJSON('POST', `${GO}/meeting/rooms/${roomCode}/end`, {}, {
    Authorization: `Bearer ${tokenA}`,
  });
  assert(endRes.status === 200, 'host end meeting ok', endRes.data);

  // 清理
  wsA.close();
  wsB.close();
  await new Promise((r) => setTimeout(r, 500));

  console.log('-----');
  console.log(`PASS=${pass} FAIL=${fail}`);
  process.exit(fail > 0 ? 1 : 0);
})().catch((err) => {
  console.error('script error:', err);
  process.exit(2);
});
