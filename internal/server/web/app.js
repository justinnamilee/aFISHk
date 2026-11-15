'use strict';
const $ = (id) => document.getElementById(id);
let csrf = '', connected = false, pc = null, lease = null, busy = false, audioBusy = false, imageURL = null;
function show(id, message, error = false) { $(id).textContent = message; $(id).classList.toggle('error', error); }
function controls() { $('fish').disabled = !connected || busy; $('capture').disabled = !connected || busy; $('audio-start').disabled = !connected || !!pc || audioBusy; $('audio-stop').disabled = !pc && !audioBusy; }
async function request(path, data) {
  const response = await fetch(path, {method:'POST', headers:{'Content-Type':'application/json', 'X-CSRF-Token':csrf}, body:JSON.stringify(data ?? {}), signal:AbortSignal.timeout(15000)});
  if (!response.ok) throw new Error(await response.text());
  return response.json();
}
async function status() {
  try { const response = await fetch('/api/status', {cache:'no-store', signal:AbortSignal.timeout(4000)}); if (!response.ok) throw new Error('Server unavailable or authentication required'); const data = await response.json(); csrf = data.csrf; connected = true; show('connection', 'Connected to desktop service'); }
  catch (error) { connected = false; show('connection', 'Disconnected · retrying', true); if (pc) await stopAudio(false); }
  controls();
}
async function action(path) {
  if (busy || !connected) return;
  busy = true; controls();
  try {
    const result = await request(path); show('feedback', result.message);
    if (path === '/api/screenshot') {
      const response = await fetch('/api/screenshot.png', {cache:'no-store', signal:AbortSignal.timeout(10000)});
      if (!response.ok) throw new Error('Capture saved, but image download failed');
      const fresh = URL.createObjectURL(await response.blob());
      $('screenshot').src = fresh; $('full-image').href = fresh; $('capture-panel').hidden = false;
      if (imageURL) URL.revokeObjectURL(imageURL); imageURL = fresh;
    }
  } catch (error) { show('feedback', `${error.message.trim()} · Actions are never automatically retried.`, true); }
  finally { busy = false; controls(); }
}
async function stopAudio(tellServer = true) {
  const old = pc; pc = null; clearInterval(lease); lease = null;
  if (old) { old.onconnectionstatechange = null; old.close(); }
  $('audio').srcObject = null;
  if (tellServer && old) { try { await request('/api/audio/stop'); } catch (_) { /* Server lease expires without heartbeats. */ } }
  show('audio-state', 'Audio stopped'); controls();
}
async function startAudio() {
  if (pc || audioBusy) return;
  audioBusy = true; controls(); show('audio-state', 'Connecting audio…');
  try {
    if (!window.RTCPeerConnection) throw new Error('This browser does not support WebRTC');
    const peer = new RTCPeerConnection({iceServers:[]}); pc = peer;
    peer.addTransceiver('audio', {direction:'recvonly'});
    peer.ontrack = (event) => { $('audio').srcObject = new MediaStream([event.track]); $('audio').play().catch(() => show('audio-state', 'Tap Play in the audio player to allow playback')); };
    peer.onconnectionstatechange = () => {
      if (pc !== peer) return;
      show('audio-state', `Audio ${peer.connectionState}`);
      if (['failed','disconnected'].includes(peer.connectionState)) stopAudio().then(() => show('audio-state','Audio disconnected · tap Listen to reconnect',true));
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
    lease = setInterval(() => request('/api/audio/keepalive').catch(async () => {await stopAudio(false); show('audio-state','Audio unavailable · check FFmpeg/PipeWire and reconnect',true);}),5000);
  } catch (error) { await stopAudio(); show('audio-state',error.message,true); }
  finally { audioBusy = false; controls(); }
}
$('fish').addEventListener('click', () => action('/api/fish'));
$('capture').addEventListener('click', () => action('/api/screenshot'));
$('audio-start').addEventListener('click',startAudio);
$('audio-stop').addEventListener('click',() => stopAudio());
window.addEventListener('offline',async () => {connected=false; await stopAudio(false); controls(); show('connection','Network disconnected',true);});
document.addEventListener('visibilitychange', () => { if (document.hidden) stopAudio(); });
status(); setInterval(status,5000);
