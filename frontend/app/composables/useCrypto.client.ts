// 浏览器端加密/编码工具（仅客户端）
// 提供 AES-GCM 加密/解密与 Base64 编解码

const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();

function getSubtle(): SubtleCrypto {
  if (typeof window === 'undefined' || !window.crypto?.subtle) {
    throw new Error('Web Crypto API 不可用');
  }
  return window.crypto.subtle;
}

export const b64encode = (bytes: Uint8Array): string => {
  if (typeof window === 'undefined') return '';
  const bin = Array.from(bytes)
    .map((b) => String.fromCharCode(b))
    .join('');
  // btoa 期望字符串为 Latin1；此处 bytes 已是 0-255 范围
  return btoa(bin);
};

export const b64decode = (b64: string): Uint8Array => {
  if (typeof window === 'undefined') return new Uint8Array();
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
};

async function importPbkdf2Key(passphrase: string): Promise<CryptoKey> {
  return await getSubtle().importKey(
    'raw',
    textEncoder.encode(passphrase),
    { name: 'PBKDF2' },
    false,
    ['deriveKey']
  );
}

async function deriveAesGcmKey(
  baseKey: CryptoKey,
  salt: Uint8Array,
  rounds: number
): Promise<CryptoKey> {
  return await getSubtle().deriveKey(
    {
      name: 'PBKDF2',
      // 使用 ArrayBuffer 以满足 BufferSource 要求
      salt: salt.buffer as ArrayBuffer,
      iterations: rounds,
      hash: 'SHA-256',
    },
    baseKey,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );
}

export async function encryptTextAesGcm(
  plainText: string,
  passphrase: string,
  options?: { rounds?: number }
): Promise<{ iv: Uint8Array; ciphertext: Uint8Array; salt: Uint8Array; rounds: number }>
{
  const subtle = getSubtle();
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const rounds = Math.max(100_000, options?.rounds ?? 120_000);

  const baseKey = await importPbkdf2Key(passphrase);
  const aesKey = await deriveAesGcmKey(baseKey, salt, rounds);
  const data = textEncoder.encode(plainText);
  const enc = await subtle.encrypt(
    { name: 'AES-GCM', iv: iv.buffer as ArrayBuffer },
    aesKey,
    data.buffer as ArrayBuffer,
  );
  return { iv, ciphertext: new Uint8Array(enc), salt, rounds };
}

export async function decryptTextAesGcm(
  ciphertext: Uint8Array,
  passphrase: string,
  salt: Uint8Array,
  iv: Uint8Array,
  rounds: number
): Promise<string> {
  const subtle = getSubtle();
  const baseKey = await importPbkdf2Key(passphrase);
  const aesKey = await deriveAesGcmKey(baseKey, salt, rounds);
  const dec = await subtle.decrypt(
    { name: 'AES-GCM', iv: iv.buffer as ArrayBuffer },
    aesKey,
    ciphertext.buffer as ArrayBuffer,
  );
  return textDecoder.decode(new Uint8Array(dec));
}


