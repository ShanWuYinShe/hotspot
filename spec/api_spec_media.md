# HotSpot API - 文件与媒体上传

> 本文档定义 HotSpot 文件与媒体上传 (`media`) 相关的 RPC 接口。

## 模块信息

- **命名空间**: `media`
- **基础路径**: `/api/v1/media/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 节点 |
|-----|------|------|
| `POST /api/v1/media/presign_upload` | 申请上传预签名（files 数组，1..N） | /api/v1 |
| `POST /api/v1/media/complete_upload` | 完成上传（upload_ids 数组，1..N） | /api/v1 |

## 设计理念

文件上传采用「先上传，后引用」的安全模式。

**核心原则**：
- **file_id 与 url 完全解耦**：`file_id` 是客户端标识符，`url` 是服务端内部存储路径
- **每次上传生成唯一 file_id**：即使上传相同文件，也会返回不同的 `file_id`
- **客户端无法判断文件是否已存在**：保护用户隐私

**安全机制**：
- 上传 API 返回 `file_id`，客户端将 `file_id` 传给业务 API
- 服务端验证：`file_id` 存在性、用户权限、分类匹配
- URL 由服务端内部解析，客户端无法伪造

**上传流程（Presigned URL 模式）**：
- 客户端直传对象存储，不占用应用服务器带宽
- 支持 S3 兼容对象存储
- 上传凭证短期有效，且绑定文件特征
- 流程：`presign_upload` 获取 `upload_url` -> 客户端直传 -> `complete_upload` 生成业务 `file_id`

## 分类限制

| 分类 | 允许的MIME类型 | 最大大小 | 说明 |
|-----|--------------|--------|------|
| `avatar` | image/jpeg, image/png, image/webp | 5MB | 用户头像 |
| `group_avatar` | image/jpeg, image/png, image/webp | 5MB | 群组头像 |
| `moment_image` | image/jpeg, image/png, image/webp, image/gif | 10MB | 生活瞬间图片 |
| `chat_image` | image/jpeg, image/png, image/webp, image/gif | 10MB | 聊天图片 |
| `chat_video` | video/mp4, video/quicktime | 100MB | 聊天视频 |
| `chat_audio` | audio/mpeg, audio/aac, audio/ogg | 20MB | 聊天语音 |
| `chat_file` | 见下方黑名单约束 | 100MB | 聊天文件 |

### chat_file 安全约束（黑名单）

`chat_file` 允许传输用户自定义文件，但为防止恶意载荷（XSS、脚本注入、可执行攻击），服务端必须拒绝以下类型（参数错误）：

| 禁止类型 | 原因 |
|---------|------|
| 可执行类（`.exe`/`.bat`/`.cmd`/`.sh`/`.app`/`.msi`/`.com`/`.scr` 等） | 防止恶意可执行文件传播 |
| 脚本/标记类（`.html`/`.htm`/`.svg`/`.js`/`.xhtml`） | 防止客户端渲染时触发 XSS / 脚本注入（`.svg` 可内嵌 JS） |
| 服务端脚本类（`.php`/`.jsp`/`.asp`/`.aspx`） | 防止误在服务端被解析执行 |

**实现要求**：
- 校验基于**文件扩展名 + MIME 类型双判定**（任一命中黑名单即拒绝），防止单纯改扩展名绕过
- MIME 类型与扩展名不一致时拒绝（如 `.png` 扩展名但 `text/html` MIME）
- 未明确禁止的类型默认允许

## 端点详情

### POST /api/v1/media/presign_upload

申请上传预签名。`files` 数组接收 1..N 个文件（单文件传长度 1 的数组即可，无独立批量端点）。

- **节点**: /api/v1

**Request**: `PresignUploadRequest`
```typescript
type MediaCategory = 'avatar' | 'group_avatar' | 'moment_image' | 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';

interface PresignFileInput {
  file_name: string;
  file_size: number;
  mime_type: string;
  category: MediaCategory;
  md5: string | null;   // 文件内容 MD5（可选，服务端用于校验完整性）
}

interface PresignUploadRequest {
  files: PresignFileInput[];  // 1..N；上限 = 批次内各类别上限的最大值（moment_image → max_moment_images；chat_* → max_chat_files；默认均为 9）
}
```

**Response**: `ApiResponse<PresignedUploads>`
```typescript
interface PresignedUpload {
  upload_id: string;
  upload_url: string;
  headers: Record<string, string>;
  expires_at: string;
}

interface PresignedUploads {
  uploads: PresignedUpload[];     // 与请求 files 一一对应且顺序一致
}
```

**参数校验**（不合法返回 10000）：category 非法、mime_type 与 category 不匹配、file_size 超限、file_name 为空或超长、md5 格式非法、files 为空数组。

**原子语义**：任一文件校验失败则整批失败（不部分预签名）；每个文件独立获得 `upload_url`，客户端可并发直传。

### [客户端操作] 执行上传

- Method: `PUT`（对象存储直传，仅发生在存储服务域名上）
- URL: `upload_url`
- Headers: `headers` from response
- Body: File Binary Data

### POST /api/v1/media/complete_upload

完成上传。`upload_ids` 数组接收 1..N 个待确认上传（单文件传长度 1 的数组）。

- **节点**: /api/v1

**Request**: `CompleteUploadRequest`
```typescript
interface CompleteUploadRequest {
  upload_ids: string[];  // 1..N；上限与 presign_upload 一致
}
```

**Response**: `ApiResponse<CompletedUploads>`
```typescript
interface UploadedMedia {
  file_id: string;
  thumbnail_url: string | null;
  file_size: number;
  mime_type: string;
  duration: number | null;
  width: number | null;
  height: number | null;
}

interface CompletedUploads {
  uploads: UploadedMedia[];     // 与请求 upload_ids 一一对应且顺序一致
}
```

**错误码**: `10300`（`upload_id` 不存在或不属于当前用户） `10400`（文件直传尚未完成，客户端应完成直传后重试）

**原子语义**：任一 `upload_id` 校验失败则整批失败，不部分完成。

**幂等语义（自然去重键）**：`upload_id` 与 `file_id` 的映射在首次 complete 成功后持久化。重复传入已完成的 `upload_ids` 返回相同的 `file_id` 与元数据（等价成功），不报错。