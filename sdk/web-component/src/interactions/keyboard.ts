export interface KeyboardCallbacks {
  onUndo: () => void;
  onRedo: () => void;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onZoomToFit: () => void;
}

export class KeyboardManager {
  private callbacks: KeyboardCallbacks;
  private boundKeyDown: (e: KeyboardEvent) => void;

  constructor(callbacks: KeyboardCallbacks) {
    this.callbacks = callbacks;
    this.boundKeyDown = this.onKeyDown.bind(this);
  }

  attach(root: ShadowRoot | HTMLElement): void {
    root.addEventListener('keydown', this.boundKeyDown as EventListener);
  }

  detach(root: ShadowRoot | HTMLElement): void {
    root.removeEventListener('keydown', this.boundKeyDown as EventListener);
  }

  private onKeyDown(e: KeyboardEvent): void {
    const tag = (e.target as HTMLElement).tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;
    const mod = e.metaKey || e.ctrlKey;

    if (mod && e.key === 'z' && !e.shiftKey) {
      e.preventDefault();
      this.callbacks.onUndo();
    } else if (mod && e.key === 'z' && e.shiftKey) {
      e.preventDefault();
      this.callbacks.onRedo();
    } else if (mod && e.key === 'y') {
      e.preventDefault();
      this.callbacks.onRedo();
    } else if (mod && e.key === '=') {
      e.preventDefault();
      this.callbacks.onZoomIn();
    } else if (mod && e.key === '-') {
      e.preventDefault();
      this.callbacks.onZoomOut();
    } else if (mod && e.key === '0') {
      e.preventDefault();
      this.callbacks.onZoomToFit();
    }
  }
}
