// Arena Basica Canvas Renderer

class ArenaRenderer {
  constructor(canvasId) {
    this.canvas = document.getElementById(canvasId);
    this.ctx = this.canvas.getContext('2d');
    this.gridSize = 10;
    this.cellSize = this.canvas.width / this.gridSize;
    this.colors = ['#38bdf8', '#f43f5e', '#a855f7', '#eab308'];
    this.frames = [];
    this.currentTick = 0;
    this.isPlaying = false;
    this.timer = null;

    this.initControls();
  }

  loadReplay(replayData) {
    this.frames = replayData.frames || [];
    this.currentTick = 0;
    document.getElementById('match-info').textContent = `Match: ${replayData.match_id || 'Sample'}`;
    this.render();
  }

  render() {
    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
    this.drawGrid();

    if (!this.frames || this.frames.length === 0) return;

    const frame = this.frames[this.currentTick];
    if (!frame) return;

    document.getElementById('tick-info').textContent = `Tick: ${this.currentTick} / ${this.frames.length - 1}`;

    const players = frame.state.players || {};
    let idx = 0;
    const playerListEl = document.getElementById('player-list');
    playerListEl.innerHTML = '';

    for (const [id, p] of Object.entries(players)) {
      const color = this.colors[idx % this.colors.length];
      idx++;

      // Draw player on canvas
      if (p.alive) {
        const cx = p.x * this.cellSize + this.cellSize / 2;
        const cy = p.y * this.cellSize + this.cellSize / 2;
        const radius = this.cellSize * 0.35;

        // Shield aura
        if (p.shielded) {
          this.ctx.beginPath();
          this.ctx.arc(cx, cy, radius + 6, 0, Math.PI * 2);
          this.ctx.strokeStyle = '#60a5fa';
          this.ctx.lineWidth = 3;
          this.ctx.stroke();
        }

        this.ctx.beginPath();
        this.ctx.arc(cx, cy, radius, 0, Math.PI * 2);
        this.ctx.fillStyle = color;
        this.ctx.fill();

        // Label
        this.ctx.fillStyle = '#ffffff';
        this.ctx.font = '10px sans-serif';
        this.ctx.textAlign = 'center';
        this.ctx.fillText(id.substring(0, 5), cx, cy + 3);
      }

      // Sidebar card
      const card = document.createElement('div');
      card.className = `player-card ${p.alive ? '' : 'dead'}`;
      card.style.borderLeftColor = color;
      card.innerHTML = `
        <strong>${id}</strong> - HP: ${p.hp} | Energy: ${p.energy} | Score: ${p.score || 0}
        <div class="hp-bar"><div class="hp-fill" style="width: ${Math.max(0, p.hp)}%;"></div></div>
      `;
      playerListEl.appendChild(card);
    }

    // Events
    const logEl = document.getElementById('event-log');
    if (frame.events && frame.events.length > 0) {
      logEl.innerHTML = frame.events.map(e => `<div>• ${e}</div>`).join('');
    }
  }

  drawGrid() {
    this.ctx.strokeStyle = '#1e293b';
    this.ctx.lineWidth = 1;

    for (let i = 0; i <= this.gridSize; i++) {
      this.ctx.beginPath();
      this.ctx.moveTo(i * this.cellSize, 0);
      this.ctx.lineTo(i * this.cellSize, this.canvas.height);
      this.ctx.stroke();

      this.ctx.beginPath();
      this.ctx.moveTo(0, i * this.cellSize);
      this.ctx.lineTo(this.canvas.width, i * this.cellSize);
      this.ctx.stroke();
    }
  }

  initControls() {
    document.getElementById('btn-play').addEventListener('click', () => {
      this.isPlaying = !this.isPlaying;
      document.getElementById('btn-play').textContent = this.isPlaying ? 'Pause' : 'Play';
      if (this.isPlaying) this.play();
    });

    document.getElementById('btn-next').addEventListener('click', () => {
      if (this.currentTick < this.frames.length - 1) {
        this.currentTick++;
        this.render();
      }
    });

    document.getElementById('btn-prev').addEventListener('click', () => {
      if (this.currentTick > 0) {
        this.currentTick--;
        this.render();
      }
    });

    document.getElementById('btn-reset').addEventListener('click', () => {
      this.currentTick = 0;
      this.render();
    });
  }

  play() {
    if (!this.isPlaying) return;
    if (this.currentTick < this.frames.length - 1) {
      this.currentTick++;
      this.render();
      setTimeout(() => this.play(), 250);
    } else {
      this.isPlaying = false;
      document.getElementById('btn-play').textContent = 'Play';
    }
  }
}

window.addEventListener('DOMContentLoaded', () => {
  window.renderer = new ArenaRenderer('arenaCanvas');
});
