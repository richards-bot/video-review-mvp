'use strict';

let cachedKey = null;

async function getKey(cekBytes) {
  if (cachedKey) return cachedKey;
  cachedKey = await crypto.subtle.importKey(
    'raw',
    cekBytes,
    { name: 'AES-GCM' },
    false,
    ['decrypt']
  );
  return cachedKey;
}

function chunkNonce(chunkIndex) {
  const nonce = new Uint8Array(12);
  // bytes 0-3 are 0x00
  const view = new DataView(nonce.buffer);
  view.setBigUint64(4, BigInt(chunkIndex), false); // big-endian
  return nonce;
}

function manifestNonce() {
  const nonce = new Uint8Array(12);
  nonce[0] = 0xFF;
  nonce[1] = 0xFF;
  nonce[2] = 0xFF;
  nonce[3] = 0xFF;
  // bytes 4-11 are 0x00
  return nonce;
}

function chunkAAD(assetId, chunkIndex) {
  return new TextEncoder().encode(`${assetId}|${chunkIndex}`);
}

function manifestAAD(assetId) {
  return new TextEncoder().encode(`${assetId}|manifest`);
}

self.addEventListener('message', async (event) => {
  const msg = event.data;

  try {
    if (msg.type === 'decrypt') {
      const { assetId, chunkIndex, ciphertext, cek } = msg;
      const key = await getKey(new Uint8Array(cek));
      const nonce = chunkNonce(chunkIndex);
      const aad = chunkAAD(assetId, chunkIndex);

      let plaintext;
      try {
        plaintext = await crypto.subtle.decrypt(
          { name: 'AES-GCM', iv: nonce, additionalData: aad },
          key,
          ciphertext
        );
      } catch (e) {
        self.postMessage({
          type: 'error',
          chunkIndex,
          message: 'chunk-auth-failed'
        });
        return;
      }

      self.postMessage(
        { type: 'result', chunkIndex, plaintext },
        [plaintext]
      );

    } else if (msg.type === 'decrypt-manifest') {
      const { assetId, ciphertext, cek } = msg;
      const key = await getKey(new Uint8Array(cek));
      const nonce = manifestNonce();
      const aad = manifestAAD(assetId);

      let plaintext;
      try {
        plaintext = await crypto.subtle.decrypt(
          { name: 'AES-GCM', iv: nonce, additionalData: aad },
          key,
          ciphertext
        );
      } catch (e) {
        self.postMessage({
          type: 'error',
          chunkIndex: -1,
          message: 'manifest-auth-failed'
        });
        return;
      }

      self.postMessage(
        { type: 'result', chunkIndex: -1, plaintext },
        [plaintext]
      );
    }
  } catch (err) {
    self.postMessage({
      type: 'error',
      chunkIndex: msg.chunkIndex ?? -1,
      message: err.message || 'unknown-worker-error'
    });
  }
});
