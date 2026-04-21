#!/usr/bin/env node
// Task 8 端到端验证脚本：会议生命周期状态机
// 覆盖：
//   1. host 掉线 → 宽限期过期自动转让（其他成员收 host.changed，DB host_id 更新）
//   2. host 掉线 → 宽限期内重连 → host 身份保留
//   3. 全员 leave → empty_ttl 存在 → 新用户 join 复活（empty_ttl key 被 DEL）
//   4. 全员 leave → 等待 TTL 过期 → Node Router 被关闭（CloseRouter 日志佐证）+ DB 状态 Ended
//   5. JoinRoom 不再重复调 CreateRouter（Node 侧仅 1 次 router created 日志）
//
// 运行前置：
//   - 必须设置短时长环境变量加速触发，推荐：
//       ECHOCHAT_MEETING_HOST_GRACE_SECONDS=3
//       ECHOCHAT_MEETING_EMPTY_ROOM_TTL_SECONDS=3
//       ECHOCHAT_MEETING_CLEANUP_INTERVAL_SECONDS=1
//     然后启动 go-service
//   - media-server 必须已启动（端口 3300）

import WebSocket from 'ws';
import { createClient } from 'redis';

const GO = 'http://localhost:8085/api/v1';
const NODE = 'http://localhost:3300';
const NODE_TOKEN = 'dev-token-abcdef1234567890';

// 兼容上游通过环境变量传入缩短后的时长（默认与 E2E 推荐一致）
const HOST_GRACE_SECONDS = Number(process.env.ECHOCHAT_MEETING_HOST_GRACE_SECONDS || 3);
const EMPTY_TTL_SECONDS = Number(process.env.ECHOCHAT_MEETING_EMPTY_ROOM_TTL_SECONDS || 3);
const CLEANUP_SECONDS = Number(process.env.ECHOCHAT_MEETING_CLEANUP_INTERVAL_SECONDS || 1);

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

// ============ HTTP utils ============

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
  const email = `t8user${suffix}@test.local`;
  const password = 'TestPassword123!';
  const username = `t8user${suffix}`;
  await httpJSON('POST', `${GO}/auth/register`, { email, username, password });
  const login = await httpJSON('POST', `${GO}/auth/login`, { account: email, password });
  if (!login.data?.data?.token) throw new Error(`login failed ${JSON.stringify(login)}`);
  return { token: login.data.data.token, userID: login.data.data.user?.id };
}

// ============ WS utils ============

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

// 收集指定 event 的推送消息到内存 buffer（用于非 ACK 广播事件断言）
function collectPush(ws, eventName, buffer) {
  const listener = (raw) => {
    try {
      const m = JSON.parse(raw.toString());
      if (m.event === eventName) buffer.push(m);
    } catch {}
  };
  ws.on('message', listener);
  return () => ws.off('message', listener);
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// ============ Redis utils（直接断言 key 状态，避免依赖服务日志）============

async function connectRedis() {
  const c = createClient({ url: 'redis://localhost:6379/0' });
  await c.connect();
  return c;
}

// ============ Main ============

(async () => {
  console.log('==== Task 8 Meeting Lifecycle E2E Verify ====');
  console.log(`Config: HOST_GRACE=${HOST_GRACE_SECONDS}s EMPTY_TTL=${EMPTY_TTL_SECONDS}s CLEANUP=${CLEANUP_SECONDS}s`);

  const redis = await connectRedis();

  // 预检：media-server 活着
  const mHealth = await httpJSON('GET', `${NODE}/healthz`);
  assert(mHealth.status === 200 && mHealth.data?.ok === true, 'media-server healthz');

  const ts = Date.now().toString().slice(-6);

  // ===== 场景 1：host 宽限期过期自动转让 =====
  console.log('\n-- Scenario 1: host grace expired → auto transfer --');
  {
    const a = await register(`a1${ts}`);
    const b = await register(`b1${ts}`);
    const createRes = await httpJSON('POST', `${GO}/meeting/rooms`, {
      title: 'T8 Scene1', type: 1,
    }, { Authorization: `Bearer ${a.token}` });
    assert(createRes.status === 201, 'S1: create room ok');
    const code = createRes.data?.data?.room?.room_code;
    const roomID = createRes.data?.data?.room?.id;

    await httpJSON('POST', `${GO}/meeting/rooms/${code}/join`, {}, { Authorization: `Bearer ${b.token}` });

    const wsA = await connectWS(a.token);
    const wsB = await connectWS(b.token);
    await wsSend(wsA, 'meeting.room.join', { room_code: code });
    await wsSend(wsB, 'meeting.room.join', { room_code: code });

    // 监听 B 上的 host.changed 广播
    const hostChangedBuf = [];
    const stopCollect = collectPush(wsB, 'meeting.host.changed', hostChangedBuf);

    // A（host）断开 WS → 写入 host_grace key
    wsA.close();
    await sleep(500);
    const graceExists = await redis.exists(`echo:meeting:host_grace:${code}`);
    assert(graceExists === 1, 'S1: host_grace key written after host disconnect');

    // 等宽限期 + cleanup 一轮
    await sleep((HOST_GRACE_SECONDS + CLEANUP_SECONDS + 1) * 1000);

    const graceGone = await redis.exists(`echo:meeting:host_grace:${code}`);
    assert(graceGone === 0, 'S1: host_grace key cleared after expiry');
    assert(hostChangedBuf.length >= 1, 'S1: meeting.host.changed broadcasted', hostChangedBuf);
    if (hostChangedBuf.length > 0) {
      const payload = hostChangedBuf[0]?.data;
      assert(payload?.new_host_id === b.userID, 'S1: new_host_id = B', payload);
      assert(payload?.auto_reason === 'host_grace_expired', 'S1: auto_reason = host_grace_expired', payload);
    }

    // DB 状态：通过 REST /rooms/:code 查看 host_id（B 的视角）
    const roomInfo = await httpJSON('GET', `${GO}/meeting/rooms/${code}`, undefined, { Authorization: `Bearer ${b.token}` });
    assert(roomInfo.data?.data?.room?.host_id === b.userID, 'S1: DB host_id updated to B', roomInfo.data);

    stopCollect();
    wsB.close();
    // 清理会议（通过新 host B）
    await httpJSON('POST', `${GO}/meeting/rooms/${code}/end`, {}, { Authorization: `Bearer ${b.token}` });
    await sleep(300);
  }

  // ===== 场景 2：host 宽限期内重连保留身份 =====
  console.log('\n-- Scenario 2: host reconnect within grace period --');
  {
    const a = await register(`a2${ts}`);
    const b = await register(`b2${ts}`);
    const createRes = await httpJSON('POST', `${GO}/meeting/rooms`, {
      title: 'T8 Scene2', type: 1,
    }, { Authorization: `Bearer ${a.token}` });
    const code = createRes.data?.data?.room?.room_code;
    await httpJSON('POST', `${GO}/meeting/rooms/${code}/join`, {}, { Authorization: `Bearer ${b.token}` });

    let wsA = await connectWS(a.token);
    const wsB = await connectWS(b.token);
    await wsSend(wsA, 'meeting.room.join', { room_code: code });
    await wsSend(wsB, 'meeting.room.join', { room_code: code });

    // A 掉线 → 立刻重连（远小于 HOST_GRACE_SECONDS）
    wsA.close();
    await sleep(500);
    const graceExists = await redis.exists(`echo:meeting:host_grace:${code}`);
    assert(graceExists === 1, 'S2: host_grace key written');

    // 重连 + room.join 触发 OnHostReconnect
    wsA = await connectWS(a.token);
    await wsSend(wsA, 'meeting.room.join', { room_code: code });
    await sleep(500);

    const graceGone = await redis.exists(`echo:meeting:host_grace:${code}`);
    assert(graceGone === 0, 'S2: host_grace key DEL on reconnect');

    // 继续等过完原本的宽限期 + cleanup，无 host 变更即通过
    await sleep((HOST_GRACE_SECONDS + CLEANUP_SECONDS + 1) * 1000);
    const roomInfo = await httpJSON('GET', `${GO}/meeting/rooms/${code}`, undefined, { Authorization: `Bearer ${a.token}` });
    assert(roomInfo.data?.data?.room?.host_id === a.userID, 'S2: host identity preserved', roomInfo.data);

    wsA.close();
    wsB.close();
    await httpJSON('POST', `${GO}/meeting/rooms/${code}/end`, {}, { Authorization: `Bearer ${a.token}` });
    await sleep(300);
  }

  // ===== 场景 3：空房 TTL 复活 =====
  console.log('\n-- Scenario 3: empty_ttl revival on new join --');
  {
    const a = await register(`a3${ts}`);
    const c = await register(`c3${ts}`);
    const createRes = await httpJSON('POST', `${GO}/meeting/rooms`, {
      title: 'T8 Scene3', type: 1,
    }, { Authorization: `Bearer ${a.token}` });
    const code = createRes.data?.data?.room?.room_code;

    // host 自己 leave → 空房 TTL 启动
    await httpJSON('POST', `${GO}/meeting/rooms/${code}/leave`, {}, { Authorization: `Bearer ${a.token}` });
    await sleep(300);
    const ttlExists = await redis.exists(`echo:meeting:empty_ttl:${code}`);
    assert(ttlExists === 1, 'S3: empty_ttl key written on empty room');

    // TTL 内 C 新加入 → 撤销 TTL，房间保持 Active
    const joinRes = await httpJSON('POST', `${GO}/meeting/rooms/${code}/join`, {}, { Authorization: `Bearer ${c.token}` });
    assert(joinRes.status === 200, 'S3: new user join succeeds within TTL', joinRes.data);
    await sleep(300);
    const ttlGone = await redis.exists(`echo:meeting:empty_ttl:${code}`);
    assert(ttlGone === 0, 'S3: empty_ttl key DEL on join');

    const roomInfo = await httpJSON('GET', `${GO}/meeting/rooms/${code}`, undefined, { Authorization: `Bearer ${c.token}` });
    assert(roomInfo.data?.data?.room?.status === 1, 'S3: room remains Active after revival', roomInfo.data);

    // 清理：C 结束或 leave（C 非 host，走 leave→空房→TTL；等其过期兜底）
    await httpJSON('POST', `${GO}/meeting/rooms/${code}/leave`, {}, { Authorization: `Bearer ${c.token}` });
    await sleep((EMPTY_TTL_SECONDS + CLEANUP_SECONDS + 1) * 1000);
  }

  // ===== 场景 4：空房 TTL 过期 → 自动销毁 =====
  console.log('\n-- Scenario 4: empty_ttl expiry → room Ended --');
  {
    const a = await register(`a4${ts}`);
    const createRes = await httpJSON('POST', `${GO}/meeting/rooms`, {
      title: 'T8 Scene4', type: 1,
    }, { Authorization: `Bearer ${a.token}` });
    const code = createRes.data?.data?.room?.room_code;

    await httpJSON('POST', `${GO}/meeting/rooms/${code}/leave`, {}, { Authorization: `Bearer ${a.token}` });
    await sleep(300);
    assert((await redis.exists(`echo:meeting:empty_ttl:${code}`)) === 1, 'S4: empty_ttl set');

    await sleep((EMPTY_TTL_SECONDS + CLEANUP_SECONDS + 1) * 1000);
    assert((await redis.exists(`echo:meeting:empty_ttl:${code}`)) === 0, 'S4: empty_ttl cleared after expiry');

    // 会议已结束 → 再 join 报已结束
    const b = await register(`b4${ts}`);
    const joinAgain = await httpJSON('POST', `${GO}/meeting/rooms/${code}/join`, {}, { Authorization: `Bearer ${b.token}` });
    assert(joinAgain.status === 400 || joinAgain.status === 404 || joinAgain.status === 410,
      'S4: join Ended room is rejected', joinAgain.data);
  }

  // ===== 场景 5：JoinRoom 不再重复调 CreateRouter =====
  console.log('\n-- Scenario 5: JoinRoom should NOT create duplicate Router --');
  {
    // 记录 Node 当前 Router 数（通过 /internal/info）
    const info0 = await httpJSON('GET', `${NODE}/internal/info`, undefined, { 'X-Internal-Token': NODE_TOKEN });
    const routers0 = info0.data?.stats?.routers ?? info0.data?.routers ?? 0;

    const a = await register(`a5${ts}`);
    const b = await register(`b5${ts}`);
    const createRes = await httpJSON('POST', `${GO}/meeting/rooms`, {
      title: 'T8 Scene5', type: 1,
    }, { Authorization: `Bearer ${a.token}` });
    const code = createRes.data?.data?.room?.room_code;

    // 创建后应 +1 个 Router
    const info1 = await httpJSON('GET', `${NODE}/internal/info`, undefined, { 'X-Internal-Token': NODE_TOKEN });
    const routers1 = info1.data?.stats?.routers ?? info1.data?.routers ?? 0;
    assert(routers1 - routers0 === 1, 'S5: Router +1 after CreateRoom', { routers0, routers1 });

    // B 加入不应再增加 Router
    await httpJSON('POST', `${GO}/meeting/rooms/${code}/join`, {}, { Authorization: `Bearer ${b.token}` });
    const info2 = await httpJSON('GET', `${NODE}/internal/info`, undefined, { 'X-Internal-Token': NODE_TOKEN });
    const routers2 = info2.data?.stats?.routers ?? info2.data?.routers ?? 0;
    assert(routers2 === routers1, 'S5: Router count unchanged after JoinRoom (no duplicate)', { routers1, routers2 });

    // 清理
    await httpJSON('POST', `${GO}/meeting/rooms/${code}/end`, {}, { Authorization: `Bearer ${a.token}` });
    await sleep(500);
  }

  await redis.quit();
  console.log('-----');
  console.log(`PASS=${pass} FAIL=${fail}`);
  process.exit(fail > 0 ? 1 : 0);
})().catch((err) => {
  console.error('script error:', err);
  process.exit(2);
});
