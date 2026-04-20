# Phase 2d 富媒体消息 Bug 修复报告

**修复日期**：2026-04-20
**验证方式**：Playwright MCP 浏览器自动化
**测试账号**：前台 `testuser1 / test123456`、管理端 `admin_test / admin123456`

---

## 一、问题概述

用户在功能测试中反馈：**图片上传、聊天发送图片报错**，页面上无法看到图片消息。经 Playwright 自动化复现后，抓取到连锁的四处缺陷，全部已修复并通过端到端验证。

---

## 二、根因分析与修复

### Bug 1：文件上传请求指向错误端口（ERR_CONNECTION_REFUSED）

- **现象**：`POST http://localhost:8080/api/v1/upload/image` 报 `ERR_CONNECTION_REFUSED`。
- **根因**：`frontend/src/api/file.js` 将 `BASE_URL` 硬编码为 `http://localhost:8080`，与实际后端端口 `:8085` 不一致。
- **修复**：复用 `@/utils/request.js` 中的统一 `BASE_URL`，开发环境自动拼接 `http://{hostname}:8085`，生产环境走相对路径，由 Nginx 反代。

```7:13:frontend/src/api/file.js
 *
 * BASE_URL 复用 @/utils/request 中的统一配置，保证与业务接口一致
 * （开发环境：http://{hostname}:8085；生产环境：相对路径由 Nginx 反代）
 */

import { useUserStore } from '@/store/user'
import { BASE_URL } from '@/utils/request'
```

### Bug 2：MinIO 存储桶默认 private，图片 URL 返回 403

- **现象**：后端返回的 `http://localhost:9000/echochat/...jpg` 被 MinIO 以 **403 Forbidden** 拒绝匿名访问；前端 `<uni-image>` 因此无法加载图片。
- **根因**：`pkg/storage/minio.go` 仅创建 bucket，未设置 Bucket Policy；MinIO 默认对象为私有，必须带签名或登录凭据才能访问。
- **修复**：在启动时为 bucket 设置 **public-read** 策略（仅 `s3:GetObject` 匿名可达，上传仍须 AccessKey/SecretKey）。

```41:58:backend/go-service/pkg/storage/minio.go
// ensureBucketPublicRead 设置 bucket 的匿名公开读策略
// 允许匿名用户通过 HTTP GET 访问 bucket 内的对象（s3:GetObject）
// 其他操作（上传、删除等）仍然需要 AccessKey/SecretKey 凭据
func ensureBucketPublicRead(ctx context.Context, client *minio.Client, bucket string) error {
	policy := fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"AWS": ["*"]},
      "Action": ["s3:GetObject"],
      "Resource": ["arn:aws:s3:::%s/*"]
    }
  ]
}`, bucket)
	return client.SetBucketPolicy(ctx, bucket, policy)
}
```

### Bug 3：MsgImage 组件布局坍缩，图片不可见

- **现象**：即便 MinIO 可匿名访问后，`bubble-media` 宽度仅 12px、`<uni-image>` 宽度 0px，图片无法可视化。
- **根因**：`MsgImage.vue` 中 `.msg-image-wrap` 为 flex 容器，但只设置了 `max-width`，未给 `.grid-single` 等宿主元素显式宽度；`.img-thumb` 使用 `width:100%` 在没有父级宽度参考时计算结果为 0，导致气泡整体塌陷。
- **修复**：给 `.grid-single / .grid-single .img-thumb` 等增加 **显式宽高**（400rpx × 300rpx 单图、156rpx × 156rpx 2 列、110rpx × 110rpx 3 列），并声明 `display: block` 以避免 inline-block 折行/塌陷。

```68:97:frontend/src/components/msg/MsgImage.vue
.grid-single {
  width: 400rpx;
}
.grid-single .img-item {
  width: 100%;
  min-width: 200rpx;
  max-width: 400rpx;
}
.grid-single .img-thumb {
  width: 400rpx;
  height: 300rpx;
  border-radius: 12rpx;
  display: block;
}
```

### Bug 4（附带优化）：H5 下文件消息丢失原始文件名

- **现象**：H5 下上传 `test-doc.txt` 后，聊天气泡显示为 `file-1776653949262`。
- **根因**：`URL.createObjectURL(file)` 生成的 blob URL 没有保留文件名，`uni.uploadFile` 把 blob 的临时名写回 `fileHeader.Filename`，后端据此生成的 `result.file_name` 丢失原名。
- **修复**：前端发送消息时优先使用用户选择的 `file.name`，仅当其为空时才回退到后端返回的 `result.file_name`。单聊 / 群聊 H5 及非 H5 分支同步修改。

```383:395:frontend/src/pages/chat/conversation.vue
          chatStore.sendFileMessage({
            conversationId: conversationId.value || 0,
            targetUserId: conversationId.value ? 0 : peerId.value,
            file: {
              url: result.url,
              // 优先使用前端原始文件名（H5 下 blob URL 会丢失原名）
              file_name: file.name || result.file_name,
              size: result.size,
              mime_type: file.type || 'application/octet-stream',
              ext: '.' + (file.name.split('.').pop() || '')
            }
          })
```

---

## 三、Playwright 端到端验证

| # | 验证步骤 | 结果 |
|---|---|---|
| 1 | 修复前：H5 上传图片，控制台报 `ERR_CONNECTION_REFUSED @ :8080` | ❌ 已复现 |
| 2 | 修复 `file.js` 后：上传请求落到 `:8085`，HTTP 200，后端返回完整 `url/thumbnail_url` | ✅ 通过 |
| 3 | 修复 MinIO 策略后：匿名 `curl http://localhost:9000/.../_thumb.jpg` → **HTTP 200** | ✅ 通过 |
| 4 | 修复 MsgImage 布局后：单聊气泡清晰显示缩略图，保持合理的气泡尺寸 | ✅ 通过 |
| 5 | 点击缩略图触发 `uni.previewImage` 弹出**原图大图预览** | ✅ 通过 |
| 6 | 文件消息显示原始文件名 `test-doc.txt`（48 B） | ✅ 通过 |
| 7 | 管理端 `/#/message/list` 列表显示图片 / 文件消息类型 | ✅ 通过 |
| 8 | 管理端消息详情弹窗：图片内联预览 + Extra 原始 JSON 正常 | ✅ 通过 |

---

## 四、受影响文件

| 文件 | 变更类型 | 说明 |
|---|---|---|
| `frontend/src/api/file.js` | 修改 | BASE_URL 改为复用 `@/utils/request` 中的统一配置 |
| `backend/go-service/pkg/storage/minio.go` | 修改 | 增加 `ensureBucketPublicRead` 启动时设置 public-read 策略 |
| `frontend/src/components/msg/MsgImage.vue` | 修改 | 为单图/多图网格增加显式宽高，修正布局坍缩 |
| `frontend/src/pages/chat/conversation.vue` | 修改 | 文件消息优先使用前端原始文件名 |
| `frontend/src/pages/group/conversation.vue` | 修改 | 文件消息优先使用前端原始文件名（H5 + 非 H5） |
| `test-report-phase2d-bugfix.md` | 新增 | 本次 Bug 修复报告 |

---

## 五、后续建议

1. **生产环境部署**：MinIO 桶在生产集群也应保留 public-read 策略，或改为 CDN 前置缓存；若需私有鉴权（如临时签名 URL），需改造后端 `buildURL` 为 `PresignedGetObject`。
2. **错误提示增强**：`uni.uploadFile` 静默失败路径建议追加详细 `uni.showToast`（当前仅在 `fail` 分支提示），方便后续端到端问题定位。
3. **文件名长度 / 特殊字符**：现在直接使用 `file.name` 传回前端展示，需在 UI 上对超长文件名做省略（后续优化点）。
