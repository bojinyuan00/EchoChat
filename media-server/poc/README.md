# mediasoup PoC Spike

Phase 2e-2 Task 0 产物，用于验证 mediasoup + Fastify + mediasoup-client 技术栈。

**这不是正式 media-server**，正式实现详见 Task 1-2，目录位于 [`../src/`](../src)（Task 1 完成时创建）。

## 快速启动

```bash
npm install
node server.mjs
```

浏览器访问 `http://localhost:3300/`，打开两个窗口各自点击「加入并推流」即可互通。

## 结论

技术栈可用性验证通过，详见 [`../docs/poc-notes.md`](../docs/poc-notes.md)。

## 目录

```
poc/
├── server.mjs            # Node 侧：Fastify + WS 信令 + mediasoup Worker/Router
├── public/
│   ├── index.html        # 浏览器侧极简 UI
│   └── client.mjs        # 浏览器侧：mediasoup-client 完整流程
├── package.json          # 依赖清单（固定版本号）
└── .gitignore
```
