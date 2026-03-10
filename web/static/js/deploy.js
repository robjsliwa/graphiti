// Deploy manager: handles deploy button, dropdown, and export actions
import { toast } from './toast.js';

export class DeployManager {
  constructor(workflowId) {
    this.workflowId = workflowId;
    this._bindEvents();
  }

  _bindEvents() {
    const deployBtn = document.getElementById('deploy-btn');
    const chevron = document.getElementById('deploy-chevron');
    const menu = document.getElementById('deploy-menu');

    if (deployBtn) {
      deployBtn.addEventListener('click', () => this.deploy('production'));
    }

    if (chevron && menu) {
      chevron.addEventListener('click', (e) => {
        e.stopPropagation();
        menu.hidden = !menu.hidden;
      });

      // Close menu on outside click
      document.addEventListener('click', (e) => {
        if (!menu.contains(e.target) && e.target !== chevron) {
          menu.hidden = true;
        }
      });

      // Menu item handlers
      menu.querySelectorAll('.deploy-menu-item').forEach(item => {
        item.addEventListener('click', () => {
          menu.hidden = true;
          const action = item.dataset.action;
          if (action === 'deploy') {
            this.deploy(item.dataset.target);
          } else if (action === 'export') {
            this.export(item.dataset.format);
          } else if (action === 'save-draft') {
            toast.info('Workflow is auto-saved');
          }
        });
      });
    }
  }

  async deploy(target) {
    const btn = document.getElementById('deploy-btn');
    if (btn) {
      btn.disabled = true;
      btn.textContent = 'Deploying...';
    }

    try {
      const resp = await fetch(`/api/workflows/${this.workflowId}/deploy`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ target }),
      });
      const result = await resp.json();

      if (result.success) {
        toast.success(`Deployed to ${target} successfully`);
        if (btn) {
          btn.classList.add('deploy-success');
          btn.textContent = '\u2713 Deployed';
          setTimeout(() => {
            btn.classList.remove('deploy-success');
            btn.textContent = 'Deploy';
            btn.disabled = false;
          }, 2000);
        }
        // Update version badge
        const badge = document.querySelector('.version-badge');
        if (badge) {
          const current = parseInt(badge.textContent.replace('v', ''));
          badge.textContent = 'v' + (current + 1);
        }
      } else {
        const msg = result.message || 'Deploy failed';
        toast.error(msg);
        if (result.validationErrors?.length) {
          result.validationErrors.forEach(ve => {
            toast.warning(`${ve.message}${ve.nodeId ? ` (node: ${ve.nodeId.substring(0, 8)})` : ''}`);
          });
          // Highlight error nodes
          result.validationErrors.forEach(ve => {
            if (ve.nodeId) {
              const el = document.querySelector(`[data-node-id="${ve.nodeId}"]`);
              if (el) el.classList.add('node-error');
            }
          });
        }
        if (btn) {
          btn.classList.add('deploy-error');
          setTimeout(() => {
            btn.classList.remove('deploy-error');
            btn.textContent = 'Deploy';
            btn.disabled = false;
          }, 2000);
        }
      }
    } catch (err) {
      console.error('Deploy error:', err);
      toast.error('Network error during deploy');
      if (btn) {
        btn.textContent = 'Deploy';
        btn.disabled = false;
      }
    }
  }

  export(format) {
    // Trigger download via hidden link
    const url = `/api/workflows/${this.workflowId}/export?format=${format}`;
    const a = document.createElement('a');
    a.href = url;
    a.download = `workflow.${format}`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    toast.info(`Exporting as ${format.toUpperCase()}`);
  }
}
