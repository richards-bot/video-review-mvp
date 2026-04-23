'use strict';

// ── Fragment parsing ─────────────────────────────────────────────────────────

function parseFragment(hash) {
  if (!hash || hash.length < 2) return null;
  const raw = hash.startsWith('#') ? hash.slice(1) : hash;
  const params = {};
  for (const part of raw.split('&')) {
    const eq = part.indexOf('=');
    if (eq === -1) continue;
    params[part.slice(0, eq)] = part.slice(eq + 1);
  }
  return params;
}

function validateParams(params) {
  if (!params) return 'missing fragment';
  if (params.v !== '1') return 'unsupported URL version';
  if (!/^[0-9a-f]{32}$/.test(params.a)) return 'invalid asset_id';
  if (!params.k || params.k.length !== 43) return 'invalid CEK length';
  if (!params.b || params.b.length === 0) return 'missing bucket';
  if (!params.r || params.r.length === 0) return 'missing region';
  return null;
}

function decodeCEK(b64url) {
  // URL-safe base64 without padding → Uint8Array
  const b64 = b64url.replace(/-/g, '+').replace(/_/g, '/');
  const padded = b64 + '='.repeat((4 - b64.length % 4) % 4);
  const bin = atob(padded);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

// ── UI helpers ───────────────────────────────────────────────────────────────

const $ = (id) => document.getElementById(id);

function showError(msg) {
  const el = $('status');
  el.textContent = msg;
  el.className = 'error';
}

function showLoading(msg) {
  const el = $('status');
  el.textContent = msg;
  el.className = 'loading';
}

function hideStatus() {
  $('status').className = 'hidden';
}

function showMeta(manifest) {
  const durationSec = Math.round(manifest.duration_ms / 1000);
  const mins = Math.floor(durationSec / 60);
  const secs = String(durationSec % 60).padStart(2, '0');
  const sizeMB = (manifest.total_bytes / (1024 * 1024)).toFixed(0);
  const date = new Date(manifest.created_at).toISOString().slice(0, 16).replace('T', ' ') + ' UTC';

  $('meta-filename').textContent = manifest.source_filename || 'video';
  $('meta-details').textContent =
    `${manifest.width}×${manifest.height} · ${mins}:${secs} · ${sizeMB} MB · Shared by ${manifest.created_by} · ${date}`;
  $('meta').classList.add('visible');
}

function updateProgress(loaded, total) {
  const pct = total > 0 ? Math.round((loaded / total) * 100) : 0;
  $('progress-fill').style.width = pct + '%';
  $('progress-text').textContent = `Loading: ${loaded} / ${total} chunks`;
  $('progress-bar').classList.add('visible');
  $('progress-text').classList.add('visible');
}

// ── Worker communication ─────────────────────────────────────────────────────

function decryptInWorker(worker, msg) {
  return new Promise((resolve, reject) => {
    const handler = (event) => {
      const r = event.data;
      const matchIndex = msg.type === 'decrypt-manifest' ? -1 : msg.chunkIndex;
      if ((r.type === 'result' || r.type === 'error') && r.chunkIndex === matchIndex) {
        worker.removeEventListener('message', handler);
        if (r.type === 'error') reject(new Error(r.message));
        else resolve(r.plaintext);
      }
    };
    worker.addEventListener('message', handler);
    const transferables = msg.ciphertext ? [msg.ciphertext] : [];
    worker.postMessage(msg, transferables);
  });
}

// ── MSE helpers ──────────────────────────────────────────────────────────────

function appendBuffer(sb, bytes) {
  return new Promise((resolve, reject) => {
    const cleanup = () => {
      sb.removeEventListener('updateend', onEnd);
      sb.removeEventListener('error', onErr);
    };
    const onEnd = () => { cleanup(); resolve(); };
    const onErr = (e) => { cleanup(); reject(e); };
    sb.addEventListener('updateend', onEnd);
    sb.addEventListener('error', onErr);
    sb.appendBuffer(bytes);
  });
}

// Simple bounded async queue
class AsyncQueue {
  constructor(maxSize = 4) {
    this._items = [];
    this._waiting = [];
    this._putWaiters = [];
    this._closed = false;
    this._maxSize = maxSize;
  }

  async put(item) {
    while (this._items.length >= this._maxSize) {
      await new Promise(resolve => this._putWaiters.push(resolve));
    }
    this._items.push(item);
    if (this._waiting.length > 0) this._waiting.shift()(this._items.shift());
  }

  async get() {
    if (this._items.length > 0) {
      const item = this._items.shift();
      if (this._putWaiters.length > 0) this._putWaiters.shift()();
      return item;
    }
    if (this._closed) return null;
    return new Promise(resolve => this._waiting.push((item) => {
      if (this._putWaiters.length > 0) this._putWaiters.shift()();
      resolve(item);
    }));
  }

  close() {
    this._closed = true;
    while (this._waiting.length > 0) this._waiting.shift()(null);
  }
}

// ── Main ─────────────────────────────────────────────────────────────────────

async function main() {
  // Feature detection
  if (!('MediaSource' in window)) {
    showError('This browser does not support the required video playback features. Please try Chrome, Firefox, or Safari 15+.');
    return;
  }
  if (!crypto.subtle || !crypto.subtle.decrypt) {
    showError('This browser does not support Web Crypto. Please use a modern browser over HTTPS.');
    return;
  }

  // iPhone detection (incomplete MSE support)
  if (/iPhone/.test(navigator.userAgent)) {
    showError('iPhone is not supported. Please use iPad, Mac, or a desktop browser.');
    return;
  }

  const params = parseFragment(window.location.hash);
  const validationError = validateParams(params);
  if (validationError) {
    showError('This link appears invalid. Please check you have the complete URL.');
    return;
  }

  const { a: assetId, k: cekB64, b: bucket, r: region } = params;

  let cekBytes;
  try {
    cekBytes = decodeCEK(cekB64);
    if (cekBytes.length !== 32) throw new Error('wrong length');
  } catch {
    showError('This link appears invalid. Please check you have the complete URL.');
    return;
  }

  // Clear fragment from URL bar immediately to avoid key staying visible
  history.replaceState(null, '', window.location.pathname);

  const s3Base = `https://${bucket}.s3.${region}.amazonaws.com/${assetId}/`;

  // Start worker
  const worker = new Worker('./decrypt-worker.js');

  showLoading('Fetching manifest…');

  // Fetch manifest
  let manifestResp;
  try {
    manifestResp = await fetch(s3Base + 'manifest.enc');
  } catch {
    showError('Unable to reach storage. Check your connection and try again.');
    worker.terminate();
    return;
  }

  if (manifestResp.status === 404) {
    showError('Content not found. The link may have expired or been withdrawn.');
    worker.terminate();
    return;
  }
  if (!manifestResp.ok) {
    showError('Unable to reach storage. Check your connection and try again.');
    worker.terminate();
    return;
  }

  const manifestCiphertext = await manifestResp.arrayBuffer();

  // Decrypt manifest in worker
  showLoading('Decrypting manifest…');
  let manifestPlaintext;
  try {
    manifestPlaintext = await decryptInWorker(worker, {
      type: 'decrypt-manifest',
      assetId,
      ciphertext: manifestCiphertext,
      cek: cekBytes.buffer
    });
  } catch (e) {
    if (e.message === 'manifest-auth-failed') {
      showError('The key in this link does not match the content. Please verify the link.');
    } else {
      showError('Unable to reach storage. Check your connection and try again.');
    }
    worker.terminate();
    return;
  }

  // Parse manifest
  let manifest;
  try {
    const text = new TextDecoder().decode(manifestPlaintext);
    manifest = JSON.parse(text);
  } catch {
    showError('The key in this link does not match the content. Please verify the link.');
    worker.terminate();
    return;
  }

  // Validate manifest
  if (manifest.version !== 1) {
    showError('This link appears invalid. Please check you have the complete URL.');
    worker.terminate();
    return;
  }
  if (manifest.asset_id !== assetId) {
    showError('The key in this link does not match the content. Please verify the link.');
    worker.terminate();
    return;
  }

  // Validate codec support
  if (!MediaSource.isTypeSupported(manifest.codec.mime)) {
    showError('Your browser cannot play this video format.');
    worker.terminate();
    return;
  }

  showMeta(manifest);
  showLoading('Buffering video…');

  const video = $('video');
  const mediaSource = new MediaSource();
  video.src = URL.createObjectURL(mediaSource);

  await new Promise(resolve => mediaSource.addEventListener('sourceopen', resolve, { once: true }));

  let sourceBuffer;
  try {
    sourceBuffer = mediaSource.addSourceBuffer(manifest.codec.mime);
  } catch {
    showError('Your browser cannot play this video format.');
    worker.terminate();
    return;
  }

  const queue = new AsyncQueue(3);
  let fetchError = null;

  // Producer: fetch + decrypt
  (async () => {
    try {
      for (let i = 0; i < manifest.chunk_count; i++) {
        const chunkInfo = manifest.chunks[i];
        const url = chunkInfo.url;

        const resp = await fetch(url);
        if (!resp.ok) throw new Error(`chunk ${i}: HTTP ${resp.status}`);
        const ciphertext = await resp.arrayBuffer();

        let plaintext;
        try {
          plaintext = await decryptInWorker(worker, {
            type: 'decrypt',
            assetId,
            chunkIndex: i,
            ciphertext,
            cek: cekBytes.buffer
          });
        } catch (e) {
          if (e.message === 'chunk-auth-failed') {
            throw new Error('CHUNK_AUTH_FAILED');
          }
          throw e;
        }

        await queue.put({ index: i, bytes: plaintext });
        updateProgress(i + 1, manifest.chunk_count);
      }
    } catch (e) {
      fetchError = e;
    } finally {
      queue.close();
    }
  })();

  // Consumer: append to SourceBuffer in order
  try {
    let appendCount = 0;
    while (true) {
      const item = await queue.get();
      if (item === null) break;

      await appendBuffer(sourceBuffer, item.bytes);
      appendCount++;

      // Memory management: trim played data every 10 chunks
      if (appendCount % 10 === 0 && video.currentTime > 30 && !sourceBuffer.updating) {
        sourceBuffer.remove(0, video.currentTime - 30);
        await new Promise(r => sourceBuffer.addEventListener('updateend', r, { once: true }));
      }

      if (appendCount === 1) {
        hideStatus();
        video.play().catch(() => {}); // may fail without user gesture; that's fine
      }
    }

    if (fetchError) {
      if (fetchError.message === 'CHUNK_AUTH_FAILED') {
        showError('Content integrity check failed. This may indicate tampering. Do not trust this content.');
      } else {
        showError('Unable to reach storage. Check your connection and try again.');
      }
      worker.terminate();
      return;
    }

    if (!sourceBuffer.updating) {
      mediaSource.endOfStream();
    } else {
      sourceBuffer.addEventListener('updateend', () => mediaSource.endOfStream(), { once: true });
    }
  } catch (e) {
    showError('Unable to reach storage. Check your connection and try again.');
    worker.terminate();
    return;
  }

  worker.terminate();
}

document.addEventListener('DOMContentLoaded', () => {
  $('security-link').addEventListener('click', (e) => {
    e.preventDefault();
    const info = $('security-info');
    info.classList.toggle('visible');
  });

  main().catch(err => {
    console.error('viewer error:', err);
    showError('An unexpected error occurred. Please try reloading.');
  });
});
