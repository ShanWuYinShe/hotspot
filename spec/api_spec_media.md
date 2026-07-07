# HotSpot API - 文件与媒体上传

> 本文档定义 HotSpot 文件与媒体上传 (`media`) 相关的 RPC 接口。

## 模块信息

- **命名空间**: `media`
- **基础路径**: `/api/v1/media/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/media/presign_upload` | 申请上传预签名 | 必须 | 无 |
| `POST /api/v1/media/batch_presign_upload` | 批量申请上传预签名 | 必须 | 无 |
| `POST /api/v1/media/complete_upload` | 完成上传 | 必须 | MUST |
| `POST /api/v1/media/complete_upload/batch` | 批量完成上传 | 必须 | 无 |

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

## 分类限制

### chat_file 安全约束（黑名单）

`chat_file` 允许传输用户自定义文件，但为防止恶意载荷（XSS、脚本注入、可执行攻击），服务端必须拒绝以下类型，返回 `10000`（参数错误）：

| 禁止类型 | 原因 |
|---------|------|
| 可执行类（`.exe`/`.bat`/`.cmd`/`.sh`/`.app`/`.msi`/`.com`/`.scr` 等） | 防止恶意可执行文件传播 |
| 脚本/标记类（`.html`/`.htm`/`.svg`/`.js`/`.xhtml`） | 防止客户端渲染时触发 XSS / 脚本注入（`.svg` 可内嵌 JS） |
| 服务端脚本类（`.php`/`.jsp`/`.asp`/`.aspx`） | 防止误在服务端被解析执行 |

**实现要求**：
- 校验基于**文件扩展名 + MIME 类型双判定**（任一命中黑名单即拒绝），防止单纯改扩展名绕过
- MIME 类型与扩展名不一致时返回 `10000`（如 `.png` 扩展名但 `text/html` MIME）
- 未明确禁止的类型默认允许

## 端点详情

### POST /api/v1/media/presign_upload

申请上传预签名。

- **认证**: 必须
- **幂等**: 无

**Request**: `PresignUploadRequest`
```typescript
type MediaCategory = 'avatar' | 'group_avatar' | 'moment_image' | 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';

interface PresignUploadRequest {
  file_name: string;
  file_size: number;
  mime_type: string;
  category: MediaCategory;
  md5?: string;
}
```

**Response**: `ApiResponse<PresignedUpload>`
```typescript
interface PresignedUpload {
  upload_id: string;
  upload_url: string;
  headers: Record<string, string>;
  expires_at: string;
}
```

**错误码**: `10000`

`10000` 触发条件（示例）：
- `category` 非法或不支持
- `mime_type` 与 `category` 不匹配
- `file_size` 超出该 `category` 的限制
- `file_name` 为空或超出约束长度
- `md5` 提供但格式非法

### [客户端操作] 执行上传

- Method: `PUT`（对象存储直传，仅发生在存储服务域名上）
- URL: `upload_url`
- Headers: `headers` from response
- Body: File Binary Data

### POST /api/v1/media/batch_presign_upload

批量申请上传预签名。适用于瞬间发布（多图）、聊天批量发文件等场景。

- **认证**: 必须
- **幂等**: 无

**说明**：
- 批量预签名是原子操作：任一文件校验失败则整批失败
- 每个文件独立获得 `upload_url`，客户端可并发上传
- 上传完成后逐个调用 `complete_upload` 确认

**Request**: `BatchPresignUploadRequest`
```typescript
interface BatchPresignUploadRequest {
  files: PresignUploadRequest[];  // 最多 9 个文件（与瞬间图片上限一致）
}

interface BatchPresignedUpload {
  uploads: PresignedUpload[];     // 与请求 files 一一对应
}
```

**Response**: `ApiResponse<BatchPresignedUpload>`

**错误码**: `10000`

`10000` 触发条件（示例）：
- `files` 为空、超过 9 项、或单项字段不合法
- 任一文件的 `category` / `mime_type` / `file_size` / `file_name` / `md5` 校验失败（批量请求原子失败）

### POST /api/v1/media/complete_upload

完成上传。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `CompleteUploadRequest`
```typescript
interface CompleteUploadRequest {
  upload_id: string;
}
```

**幂等行为**（遵循 `api_spec_base.md` 幂等契约，Key 绑定 `(user_id + endpoint + key)`）：
- 相同 `Idempotency-Key` 的重试：直接返回首次成功的完整响应（缓存的 `file_id`）
- **不同 `Idempotency-Key` 但同一 `upload_id` 已完成上传**：返回**成功**（缓存的 `file_id`），不报错——这是幂等成功语义，与 base.md 一致。`upload_id` 完成上传后即不可重复完成，重提只返回已有结果
- `upload_id` 与 `Idempotency-Key` 不绑定：幂等键防的是"网络重试导致重复副作用"，不作为上传资源的占用锁

**Response**: `ApiResponse<UploadedMedia>`
```typescript
interface UploadedMedia {
  file_id: string;
  thumbnail_url?: string;
  file_size: number;
  mime_type: string;
  duration?: number;
  width?: number;
  height?: number;
}
```

**错误码**: `10000` `10300` `10400`

**错误码说明**：
- `10300`：`upload_id` 不存在（如恶意请求发送了不存在的 upload_id）
- `10400`：业务冲突——文件直传尚未完成（对象存储上传未到位），客户端应完成直传后重试

### POST /api/v1/media/complete_upload/batch

批量完成上传。适用于瞬间发布（多图）、聊天批量发文件等场景——与 `batch_presign_upload` 配对，消除发图场景下逐个调用 `complete_upload` 的 N 次请求。

- **认证**: 必须
- **幂等**: 无

**说明**：
- 与 `batch_presign_upload` 对称的原子操作：任一 `upload_id` 校验失败则整批失败，不部分完成
- 响应的 `uploads` 与请求 `upload_ids` **一一对应且顺序一致**
- 幂等不强制 MUST（避免批量场景下单个 key 语义混乱）；客户端对失败的整批可直接重试（服务端对已完成的 `upload_id` 返回既有结果），或拆分为单文件 `complete_upload` 精细处理

**Request**: `BatchCompleteUploadRequest`
```typescript
interface BatchCompleteUploadRequest {
  upload_ids: string[];  // 最多 9 个（与 batch_presign_upload 上限一致）
}
```

**Response**: `ApiResponse<BatchCompleteUpload>`
```typescript
interface BatchCompleteUpload {
  uploads: UploadedMedia[];     // 与请求 upload_ids 一一对应、顺序一致
}
```

**错误码**: `10000` `10300` `10400`

**错误码说明**：沿用 `complete_upload` 的错误语义（原子失败）：
- `10000`：`upload_ids` 为空、超过 9 项
- `10300`：任一 `upload_id` 不存在
- `10400`：任一文件直传尚未完成
