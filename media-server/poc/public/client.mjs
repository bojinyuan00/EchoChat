// PoC 浏览器端：mediasoup-client + 原生 WebSocket
// 通过 esm.sh CDN 加载 mediasoup-client，保持 PoC 零构建步骤。
// 正式 media-server 落地时会改为 frontend 侧 npm 依赖 + vite 打包。
// esm.sh 自动解析到最新 3.x（本 PoC 验证时为 3.19.0）；正式 media-server 落地会用 npm + vite 打包
import { Device } from 'https://esm.sh/mediasoup-client@3';

const logEl = document.getElementById('log');
const statusEl = document.getElementById('status');
const videosEl = document.getElementById('videos');
const btnJoin = document.getElementById('btnJoin');
const btnLeave = document.getElementById('btnLeave');

function log(msg, level = 'info') {
  const ts = new Date().toLocaleTimeString();
  const line = document.createElement('div');
  line.className = `log-${level}`;
  line.textContent = `[${ts}] ${msg}`;
  logEl.appendChild(line);
  logEl.scrollTop = logEl.scrollHeight;
  console[level === 'err' ? 'error' : 'log'](msg);
}

// ---------------- WS 信令包装（Promise-based） ----------------

class SignalClient {
  constructor(url) {
    this.url = url;
    this.ws = null;
    this.pending = new Map(); // reqId → {resolve, reject}
    this.listeners = new Map();
  }

  connect() {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(this.url);
      this.ws.onopen = () => resolve();
      this.ws.onerror = (e) => reject(e);
      this.ws.onclose = () => {
        this.emit('__close__', {});
      };
      this.ws.onmessage = (ev) => this._onMessage(ev);
    });
  }

  _onMessage(ev) {
    let msg;
    try { msg = JSON.parse(ev.data); } catch { return; }
    const { type, data, reqId } = msg;
    if (reqId && this.pending.has(reqId)) {
      const { resolve, reject } = this.pending.get(reqId);
      this.pending.delete(reqId);
      if (type === 'error') reject(new Error(data?.message || 'signal error'));
      else resolve(data);
      return;
    }
    this.emit(type, data);
  }

  request(type, data = {}) {
    const reqId = Math.random().toString(36).slice(2);
    return new Promise((resolve, reject) => {
      this.pending.set(reqId, { resolve, reject });
      this.ws.send(JSON.stringify({ type, data, reqId }));
      setTimeout(() => {
        if (this.pending.has(reqId)) {
          this.pending.delete(reqId);
          reject(new Error(`request ${type} timeout`));
        }
      }, 10_000);
    });
  }

  on(type, fn) {
    if (!this.listeners.has(type)) this.listeners.set(type, new Set());
    this.listeners.get(type).add(fn);
  }

  emit(type, data) {
    const set = this.listeners.get(type);
    if (set) for (const fn of set) fn(data);
  }

  close() {
    if (this.ws) this.ws.close();
  }
}

// ---------------- App 状态 ----------------

const state = {
  signal: null,
  device: null,
  sendTransport: null,
  recvTransport: null,
  myPeerId: null,
  localStream: null,
  producers: { audio: null, video: null },
  consumers: new Map(), // producerId → {consumer, peerId}
};

function setStatus(text) {
  statusEl.textContent = text;
}

function addVideoTile(id, stream, label, isSelf = false) {
  let box = document.getElementById(`tile-${id}`);
  if (!box) {
    box = document.createElement('div');
    box.id = `tile-${id}`;
    box.className = 'video-box' + (isSelf ? ' self' : '');
    const v = document.createElement('video');
    v.autoplay = true;
    v.playsInline = true;
    if (isSelf) v.muted = true;
    v.srcObject = stream;
    box.appendChild(v);
    const lab = document.createElement('div');
    lab.className = 'label';
    lab.textContent = label;
    box.appendChild(lab);
    videosEl.appendChild(box);
  } else {
    const v = box.querySelector('video');
    // 将新的 track 合并进已有 stream
    for (const t of stream.getTracks()) {
      if (!v.srcObject) v.srcObject = new MediaStream();
      v.srcObject.addTrack(t);
    }
  }
}

function removeVideoTile(id) {
  const box = document.getElementById(`tile-${id}`);
  if (box) box.remove();
}

// ---------------- 关键流程 ----------------

async function join() {
  btnJoin.disabled = true;
  setStatus('connecting WS...');
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  const signal = new SignalClient(`${proto}://${location.host}/ws`);
  state.signal = signal;

  signal.on('welcome', ({ peerId }) => {
    state.myPeerId = peerId;
    log(`WS connected as peer ${peerId.slice(0, 8)}`, 'ok');
  });

  signal.on('newProducer', async ({ peerId, producerId, kind }) => {
    log(`remote producer: peer=${peerId.slice(0, 8)} kind=${kind}`, 'info');
    await consumeRemote(peerId, producerId, kind);
  });

  signal.on('peerLeft', ({ peerId }) => {
    log(`peer left: ${peerId.slice(0, 8)}`, 'info');
    removeVideoTile(peerId);
    // 清理该 peer 的 consumers
    for (const [pid, entry] of state.consumers) {
      if (entry.peerId === peerId) {
        entry.consumer.close();
        state.consumers.delete(pid);
      }
    }
  });

  signal.on('consumerClosed', ({ consumerId }) => {
    for (const [pid, entry] of state.consumers) {
      if (entry.consumer.id === consumerId) {
        entry.consumer.close();
        state.consumers.delete(pid);
      }
    }
  });

  signal.on('__close__', () => {
    setStatus('disconnected');
    log('WS closed', 'err');
  });

  await signal.connect();

  // 1. Device
  setStatus('loading device...');
  const rtpCapabilities = await signal.request('getRtpCapabilities');
  const device = new Device();
  await device.load({ routerRtpCapabilities: rtpCapabilities });
  state.device = device;
  log('mediasoup Device loaded', 'ok');

  // 2. getUserMedia
  setStatus('requesting media...');
  let localStream;
  try {
    localStream = await navigator.mediaDevices.getUserMedia({
      audio: true,
      video: { width: { ideal: 640 }, height: { ideal: 360 } },
    });
  } catch (e) {
    log(`getUserMedia failed: ${e.message}`, 'err');
    setStatus('media denied');
    btnJoin.disabled = false;
    return;
  }
  state.localStream = localStream;
  addVideoTile('local', localStream, '本地（你自己）', true);

  // 3. SendTransport
  setStatus('creating send transport...');
  const sendInfo = await signal.request('createTransport', { direction: 'send' });
  const sendTransport = device.createSendTransport({
    id: sendInfo.id,
    iceParameters: sendInfo.iceParameters,
    iceCandidates: sendInfo.iceCandidates,
    dtlsParameters: sendInfo.dtlsParameters,
  });
  state.sendTransport = sendTransport;

  sendTransport.on('connect', ({ dtlsParameters }, cb, errb) => {
    signal
      .request('connectTransport', { transportId: sendTransport.id, dtlsParameters })
      .then(() => cb())
      .catch(errb);
  });

  sendTransport.on('produce', ({ kind, rtpParameters }, cb, errb) => {
    signal
      .request('produce', { transportId: sendTransport.id, kind, rtpParameters })
      .then(({ producerId }) => cb({ id: producerId }))
      .catch(errb);
  });

  sendTransport.on('connectionstatechange', (s) => {
    log(`sendTransport ICE: ${s}`, s === 'failed' ? 'err' : 'info');
  });

  // 4. produce audio + video
  const audioTrack = localStream.getAudioTracks()[0];
  const videoTrack = localStream.getVideoTracks()[0];
  if (audioTrack) {
    state.producers.audio = await sendTransport.produce({ track: audioTrack });
    log(`local audio producer: ${state.producers.audio.id.slice(0, 8)}`, 'ok');
  }
  if (videoTrack) {
    state.producers.video = await sendTransport.produce({
      track: videoTrack,
      encodings: [
        { maxBitrate: 150_000, scaleResolutionDownBy: 4 },
        { maxBitrate: 400_000, scaleResolutionDownBy: 2 },
        { maxBitrate: 1_000_000 },
      ],
      codecOptions: { videoGoogleStartBitrate: 1000 },
    });
    log(`local video producer: ${state.producers.video.id.slice(0, 8)}`, 'ok');
  }

  // 5. RecvTransport
  setStatus('creating recv transport...');
  const recvInfo = await signal.request('createTransport', { direction: 'recv' });
  const recvTransport = device.createRecvTransport({
    id: recvInfo.id,
    iceParameters: recvInfo.iceParameters,
    iceCandidates: recvInfo.iceCandidates,
    dtlsParameters: recvInfo.dtlsParameters,
  });
  state.recvTransport = recvTransport;

  recvTransport.on('connect', ({ dtlsParameters }, cb, errb) => {
    signal
      .request('connectTransport', { transportId: recvTransport.id, dtlsParameters })
      .then(() => cb())
      .catch(errb);
  });

  recvTransport.on('connectionstatechange', (s) => {
    log(`recvTransport ICE: ${s}`, s === 'failed' ? 'err' : 'info');
  });

  // 6. 拉取当前已在房间的其他 peers
  const { items } = await signal.request('getPeers');
  log(`existing remote producers: ${items.length}`, 'info');
  for (const it of items) {
    await consumeRemote(it.peerId, it.producerId, it.kind);
  }

  setStatus('in room');
  btnLeave.disabled = false;
}

async function consumeRemote(peerId, producerId, kind) {
  if (!state.recvTransport) return;
  try {
    const data = await state.signal.request('consume', {
      transportId: state.recvTransport.id,
      producerId,
      rtpCapabilities: state.device.rtpCapabilities,
    });
    const consumer = await state.recvTransport.consume({
      id: data.id,
      producerId: data.producerId,
      kind: data.kind,
      rtpParameters: data.rtpParameters,
    });
    await state.signal.request('resumeConsumer', { consumerId: consumer.id });
    state.consumers.set(producerId, { consumer, peerId });

    const stream = new MediaStream([consumer.track]);
    addVideoTile(peerId, stream, `远端 ${peerId.slice(0, 8)} · ${kind}`);
    log(`consuming ${kind} from ${peerId.slice(0, 8)}`, 'ok');

    consumer.on('transportclose', () => state.consumers.delete(producerId));
  } catch (e) {
    log(`consume failed: ${e.message}`, 'err');
  }
}

async function leave() {
  btnLeave.disabled = true;
  setStatus('leaving...');
  try {
    for (const { consumer } of state.consumers.values()) consumer.close();
    state.consumers.clear();
    if (state.producers.audio) state.producers.audio.close();
    if (state.producers.video) state.producers.video.close();
    if (state.sendTransport) state.sendTransport.close();
    if (state.recvTransport) state.recvTransport.close();
    if (state.localStream) state.localStream.getTracks().forEach((t) => t.stop());
    if (state.signal) state.signal.close();
  } catch (e) {
    log(`leave error: ${e.message}`, 'err');
  }
  videosEl.innerHTML = '';
  setStatus('idle');
  btnJoin.disabled = false;
}

btnJoin.addEventListener('click', () => {
  join().catch((e) => {
    log(`join error: ${e.message}`, 'err');
    setStatus('error');
    btnJoin.disabled = false;
  });
});
btnLeave.addEventListener('click', () => leave());

log('PoC client ready. 点击「加入并推流」，授权摄像头/麦克风后可与其他窗口互通。', 'info');
