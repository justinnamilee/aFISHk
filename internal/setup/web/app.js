'use strict';
const $ = (id) => document.getElementById(id);
let csrf = '';
let platform = '';
async function request(path, data) {
  const response = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf },
    body: JSON.stringify(data || {}),
  });
  if (!response.ok) throw new Error(await response.text());
  return response.json();
}
function message(text) {
  $('message').textContent = text;
}
let suggestedHost = '';
let suggestedOrigin = '';
function addressChanged() {
  const listen = $('ip').value.trim();
  const host = listen && listen !== '0.0.0.0' ? listen : suggestedHost;
  const origin = host ? `http://${host}:${$('port').value}` : '';
  if ($('origin').value === suggestedOrigin) $('origin').value = origin;
  if ($('audio_ip').value === suggestedHost) $('audio_ip').value = host;
  suggestedHost = host;
  suggestedOrigin = origin;
}
for (const button of document.querySelectorAll('[data-record]')) {
  const field = $(button.dataset.record);
  let recording = false;
  function finish() {
    recording = false;
    button.textContent = 'Record ' + field.labels[0].textContent.trim().toLowerCase();
  }
  button.addEventListener('click', () => {
    recording = true;
    button.textContent = 'Press a key…';
  });
  button.addEventListener('blur', finish);
  button.addEventListener('keydown', (event) => {
    if (!recording || event.key === 'Tab') return;
    event.preventDefault();
    if (event.repeat || ['Control', 'Shift'].includes(event.key)) return;
    const base = event.code.replace(/^(?:Key(?=[A-Z]$)|Digit(?=[0-9]$))/, '');
    if (event.metaKey || event.altKey || !base || base === 'Unidentified') {
      message('Choose a key with optional Ctrl or Shift.');
      return;
    }
    field.value = [event.ctrlKey && 'Ctrl', event.shiftKey && 'Shift', base].filter(Boolean).join(
      '+',
    );
    finish();
    field.focus();
  });
}
$('ip').addEventListener('change', addressChanged);
$('port').addEventListener('change', addressChanged);
$('detect').addEventListener('click', async () => {
  $('detect').disabled = true;
  $('save').disabled = true;
  let left = 5;
  message('Switch to WoW now — 5 seconds');
  const countdown = setInterval(() => {
    left--;
    message(left > 0 ? `Keep WoW focused — ${left} seconds` : 'Reading the focused game…');
  }, 1000);
  try {
    const target = await request('/api/detect');
    $('target').textContent = `${target.title} · ${target.class}${
      target.executable ? ` · ${target.executable}` : ''
    }`;
    $('save').disabled = false;
    clearInterval(countdown);
    message('Game identified. You can return to this page.');
  } catch (err) {
    clearInterval(countdown);
    message(err.message);
  } finally {
    $('detect').disabled = false;
  }
});
$('settings').addEventListener('submit', async (event) => {
  event.preventDefault();
  $('save').disabled = true;
  try {
    const data = await request('/api/save', {
      lure_key: $('lure_key').value,
      fish_key: $('fish_key').value,
      listen: `${$('ip').value}:${$('port').value}`,
      origin: $('origin').value,
      audio_ip: $('audio_ip').value,
      pulse_source: $('pulse_source').value,
      tls: $('tls').value,
      cert: $('cert').value,
      key: $('key').value,
    });
    $('setup').hidden = true;
    $('complete').hidden = false;
    $('token').value = data.token;
    $('companion').href = data.origin;
    $('companion').textContent = data.origin;
    if (data.qr) {
      $('qr').src = '/api/qr';
      $('scan').hidden = false;
    }
    $('next').textContent = platform === 'windows'
      ? 'Click Finish setup, then open aFISHk from the Start Menu if it is not already running. Keep WoW focused when fishing.'
      : 'Click Finish setup, then start your installed user service: systemctl --user start afishk';
    message('Settings saved. This page will close its local setup connection when you finish.');
  } catch (err) {
    message(err.message);
    $('save').disabled = false;
  }
});
$('copy').addEventListener('click', async () => {
  try {
    await navigator.clipboard.writeText($('token').value);
    message('Password copied.');
  } catch {
    message('Use Show password and copy it manually.');
  }
});
$('reveal').addEventListener('click', () => {
  $('token').type = $('token').type === 'password' ? 'text' : 'password';
});
$('finish').addEventListener('click', async () => {
  try {
    await request('/api/finish');
    $('finish').disabled = true;
    message('Setup finished. You can close this tab after saving the password.');
  } catch (err) {
    message(err.message);
  }
});
(async () => {
  try {
    const response = await fetch('/api/init');
    if (!response.ok) throw new Error('Cannot open local setup');
    const data = await response.json();
    csrf = data.csrf;
    platform = data.platform;
    for (const address of data.addresses) {
      const option = document.createElement('option');
      option.value = address;
      $('addresses').appendChild(option);
    }
    $('ip').value = data.addresses[0] || '127.0.0.1';
    addressChanged();
    if (data.overwrite) {
      const settings = data.settings;
      if (settings) {
        const split = settings.listen.lastIndexOf(':');
        $('ip').value = settings.listen.slice(0, split);
        $('port').value = settings.listen.slice(split + 1);
        addressChanged();
        for (const field of ['origin', 'audio_ip', 'fish_key', 'lure_key', 'tls', 'cert', 'key']) {
          if (settings[field]) $(field).value = settings[field];
        }
      }
      $('save-note').textContent =
        'Saving replaces your current configuration and keeps your browser password.';
      $('save').textContent = 'Overwrite configuration';
    }
    $('pulse_source').value = data.pulse_source;
    $('pulse_label').hidden = platform !== 'linux';
  } catch (err) {
    message(err.message);
  }
})();
