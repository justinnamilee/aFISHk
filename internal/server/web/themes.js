'use strict';
(() => {
  const themes = [
    'classic',
    'dark-mode',
    'sea-glass',
    'blue-hour',
    'driftwood',
    'kelp-forest',
    'porcelain',
    'copper-tide',
    'lavender-mist',
    'open-ocean',
    'rose-quartz',
    'inkstone',
    'lemon-grove',
    'glacier',
    'sunset-pier',
    'arcade-reef',
    'acid-lagoon',
    'bubblegum-surf',
  ];
  const key = 'afishk.theme';
  const button = document.getElementById('theme-cycle');
  const palette = document.getElementById('theme-palette');
  const chrome = document.querySelector('meta[name="theme-color"]');
  let current = 0;

  function updateChrome() {
    chrome.content = getComputedStyle(document.documentElement).getPropertyValue('--page').trim();
  }
  function select(index) {
    current = index;
    const name = themes[index];
    if (index === 0) palette.removeAttribute('href');
    else palette.href = `/themes/${name}.css`;
    const label = `Next theme (current: ${name.replaceAll('-', ' ')})`;
    button.title = label;
    button.setAttribute('aria-label', label);
    if (index === 0) updateChrome();
  }
  palette.addEventListener('load', updateChrome);
  try {
    current = Math.max(0, themes.indexOf(localStorage.getItem(key)));
  } catch {
    // Storage can be unavailable in restricted browsing contexts.
  }
  select(current);
  button.addEventListener('click', () => {
    select((current + 1) % themes.length);
    try {
      localStorage.setItem(key, themes[current]);
    } catch {
      // The theme still applies for this visit.
    }
  });
})();
