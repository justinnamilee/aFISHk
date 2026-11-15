'use strict';
const $ = (id) => document.getElementById(id);
let csrf = '', connected = false, pc = null, lease = null, busy = false, audioBusy = false, imageURL = null;
const LOG_LIMIT = 100;
let logDescending = true;
function log(message, error = false) {
  const container = $('activity-log');
  const follow = logDescending ? container.scrollTop <= 8 : container.scrollHeight - container.scrollTop - container.clientHeight <= 8;
  const previousTop = container.scrollTop;
  const previousHeight = container.scrollHeight;
  const entry = document.createElement('p');
  entry.className = error ? 'log-entry error' : 'log-entry';
  const now = new Date();
  const time = document.createElement('time');
  time.dateTime = now.toISOString();
  time.textContent = now.toLocaleTimeString([], {hour12: false});
  entry.append(time, document.createTextNode(` ${error ? 'err' : 'inf'}: ${message}`));
  if (logDescending) container.prepend(entry);
  else container.append(entry);
  const insertedHeight = container.scrollHeight - previousHeight;
  const beforeTrim = container.scrollHeight;
  // Keep the most recent entries regardless of the selected display order.
  while (container.childElementCount > LOG_LIMIT) {
    (logDescending ? container.lastElementChild : container.firstElementChild).remove();
  }
  // Follow the newest edge unless the user is reading older entries.
  if (follow) container.scrollTop = logDescending ? 0 : container.scrollHeight;
  else container.scrollTop = previousTop + (logDescending ? insertedHeight : container.scrollHeight - beforeTrim);
}
function toggleLogSort() {
  logDescending = !logDescending;
  const container = $('activity-log');
  container.replaceChildren(...Array.from(container.children).reverse());
  container.scrollTop = logDescending ? 0 : container.scrollHeight;
  const button = $('log-sort');
  button.firstElementChild.textContent = logDescending ? '▼' : '▲';
  const label = logDescending ? 'Newest first. Sort oldest first' : 'Oldest first. Sort newest first';
  button.setAttribute('aria-label', label);
  button.title = label;
}
function showConnection(message, error = false) {
  const indicator = $('connection');
  const state = connected ? 'connected' : 'disconnected';
  const changed = indicator.dataset.state !== state;
  indicator.dataset.state = state;
  indicator.title = message;
  indicator.firstElementChild.textContent = message;
  if (changed) log(message, error);
}
function controls() { $('fish').disabled = !connected || busy; $('capture').disabled = !connected || busy; $('audio-start').disabled = !connected || !!pc || audioBusy; $('audio-stop').disabled = !pc && !audioBusy; }
async function request(path, data) {
  const response = await fetch(path, {method:'POST', headers:{'Content-Type':'application/json', 'X-CSRF-Token':csrf}, body:JSON.stringify(data ?? {}), signal:AbortSignal.timeout(15000)});
  if (!response.ok) throw new Error(await response.text());
  return response.json();
}
async function status() {
  try { const response = await fetch('/api/status', {cache:'no-store', signal:AbortSignal.timeout(4000)}); if (!response.ok) throw new Error('Server unavailable or authentication required'); const data = await response.json(); csrf = data.csrf; connected = true; showConnection('Connected to desktop service'); }
  catch (error) { connected = false; showConnection('Disconnected · retrying', true); if (pc) await stopAudio(false); }
  controls();
}
async function action(path) {
  if (busy || !connected) return;
  busy = true; controls();
  try {
    const result = await request(path); log(result.message);
    if (path === '/api/screenshot') {
      const response = await fetch('/api/screenshot.png', {cache:'no-store', signal:AbortSignal.timeout(10000)});
      if (!response.ok) throw new Error('Capture saved, but image download failed');
      const fresh = URL.createObjectURL(await response.blob());
      $('screenshot').src = fresh; $('full-image').href = fresh; $('capture-panel').hidden = false;
      if (imageURL) URL.revokeObjectURL(imageURL); imageURL = fresh;
    }
  } catch (error) { log(error.message.trim(), true); }
  finally { busy = false; controls(); }
}
async function stopAudio(tellServer = true) {
  const old = pc; const wasActive = !!old || audioBusy; pc = null; clearInterval(lease); lease = null;
  if (old) { old.onconnectionstatechange = null; old.close(); }
  $('audio').srcObject = null;
  if (tellServer && old) { try { await request('/api/audio/stop'); } catch (_) { /* Server lease expires without heartbeats. */ } }
  if (wasActive) log('Audio stopped'); controls();
}
async function startAudio() {
  if (pc || audioBusy) return;
  audioBusy = true; controls(); log('Connecting audio…');
  try {
    if (!window.RTCPeerConnection) throw new Error('This browser does not support WebRTC');
    const peer = new RTCPeerConnection({iceServers:[]}); pc = peer;
    peer.addTransceiver('audio', {direction:'recvonly'});
    peer.ontrack = (event) => {
      if (pc !== peer) return;
      // Give Wi-Fi/VPN packet jitter some headroom on mobile. This is a hint;
      // unsupported browsers retain their own adaptive buffer.
      if ('jitterBufferTarget' in event.receiver) {
        try { event.receiver.jitterBufferTarget = 120; } catch (_) { /* Optional browser hint. */ }
      }
      $('audio').srcObject = new MediaStream([event.track]);
      $('audio').play().catch(() => log('Tap Play in the audio player to allow playback'));
    };
    peer.onconnectionstatechange = () => {
      if (pc !== peer) return;
      log(`Audio ${peer.connectionState}`);
      if (['failed','disconnected'].includes(peer.connectionState)) stopAudio().then(() => log('Audio disconnected · tap Listen to reconnect',true));
    };
    await peer.setLocalDescription(await peer.createOffer());
    await new Promise((resolve,reject) => {
      if (peer.iceGatheringState === 'complete') return resolve();
      const timer = setTimeout(() => reject(new Error('Browser ICE gathering timed out')),8000);
      peer.onicegatheringstatechange = () => { if (peer.iceGatheringState === 'complete') { clearTimeout(timer); resolve(); } };
    });
    if (pc !== peer) return;
    const answer = await request('/api/audio/offer',peer.localDescription);
    if (pc !== peer) { await request('/api/audio/stop'); return; }
    await peer.setRemoteDescription(answer);
    lease = setInterval(() => request('/api/audio/keepalive').catch(async () => {await stopAudio(false); log('Audio unavailable · check FFmpeg/PipeWire and reconnect',true);}),5000);
  } catch (error) { await stopAudio(); log(error.message,true); }
  finally { audioBusy = false; controls(); }
}
$('fish').addEventListener('click', () => action('/api/fish'));
$('capture').addEventListener('click', () => action('/api/screenshot'));
$('audio-start').addEventListener('click',startAudio);
$('audio-stop').addEventListener('click',() => stopAudio());
$('log-sort').addEventListener('click', toggleLogSort);
window.addEventListener('offline',async () => {connected=false; await stopAudio(false); controls(); showConnection('Network disconnected',true);});
document.addEventListener('visibilitychange', () => { if (document.hidden) stopAudio(); });
log('Prepare your character and leave WoW focused on your PC.');
status(); setInterval(status,5000);
