// Toast notification system
// Usage: toast.error('message'), toast.warning('message'), toast.success('message'), toast.info('message')

const ICONS = {
  error: '\u2716',    // ✖
  warning: '\u26A0',  // ⚠
  success: '\u2714',  // ✔
  info: '\u2139',     // ℹ
};

const DURATIONS = { error: 6000, warning: 5000, success: 3000, info: 4000 };

let container = null;

function ensureContainer() {
  if (container && document.body.contains(container)) return container;
  container = document.createElement('div');
  container.className = 'toast-container';
  container.setAttribute('aria-live', 'polite');
  document.body.appendChild(container);
  return container;
}

function show(type, message, duration) {
  const el = document.createElement('div');
  el.className = `toast toast-${type}`;
  el.setAttribute('role', 'alert');

  const icon = document.createElement('span');
  icon.className = 'toast-icon';
  icon.textContent = ICONS[type] || ICONS.info;
  el.appendChild(icon);

  const text = document.createElement('span');
  text.className = 'toast-message';
  text.textContent = message;
  el.appendChild(text);

  const close = document.createElement('button');
  close.className = 'toast-close';
  close.textContent = '\u00D7'; // ×
  close.setAttribute('aria-label', 'Dismiss');
  close.addEventListener('click', () => dismiss(el));
  el.appendChild(close);

  ensureContainer().appendChild(el);

  // Trigger enter animation on next frame
  requestAnimationFrame(() => el.classList.add('toast-visible'));

  const ms = duration || DURATIONS[type] || 4000;
  const timer = setTimeout(() => dismiss(el), ms);
  el._timer = timer;

  // Pause auto-dismiss on hover
  el.addEventListener('mouseenter', () => clearTimeout(el._timer));
  el.addEventListener('mouseleave', () => {
    el._timer = setTimeout(() => dismiss(el), 2000);
  });

  return el;
}

function dismiss(el) {
  if (!el || !el.parentNode) return;
  clearTimeout(el._timer);
  el.classList.remove('toast-visible');
  el.classList.add('toast-exit');
  el.addEventListener('animationend', () => el.remove(), { once: true });
  // Fallback if animation doesn't fire
  setTimeout(() => el.remove(), 400);
}

export const toast = {
  error: (msg, duration) => show('error', msg, duration),
  warning: (msg, duration) => show('warning', msg, duration),
  success: (msg, duration) => show('success', msg, duration),
  info: (msg, duration) => show('info', msg, duration),
};
