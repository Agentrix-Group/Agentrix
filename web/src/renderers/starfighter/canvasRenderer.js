export const PLAYER_COLORS = ['#2f8aa6', '#e07a67'];
export const WORLD_WIDTH = 2000;
export const WORLD_HEIGHT = 1000;

export function drawStarfighterArena(canvas, frame, arenaConfig = {}) {
  if (!canvas) return;
  const ctx = canvas.getContext('2d');
  if (!ctx) return;

  const width = canvas.width;
  const height = canvas.height;
  const snapshot = frame?.public_snapshot || {};
  const arenaWidth = Number(arenaConfig.arena_width || arenaConfig.width || snapshot.arena?.width || WORLD_WIDTH);
  const arenaHeight = Number(arenaConfig.arena_height || arenaConfig.height || snapshot.arena?.height || WORLD_HEIGHT);

  ctx.clearRect(0, 0, width, height);

  // Deep space background gradient
  const gradient = ctx.createLinearGradient(0, 0, width, height);
  gradient.addColorStop(0, '#0a0e17');
  gradient.addColorStop(0.5, '#0e1626');
  gradient.addColorStop(1, '#070b12');
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, width, height);

  // Background starfield
  ctx.fillStyle = '#7ca8c4';
  for (let i = 0; i < 60; i += 1) {
    const x = (i * 137 + 41) % width;
    const y = (i * 73 + 29) % height;
    ctx.globalAlpha = 0.2 + (i % 4) * 0.18;
    ctx.beginPath();
    ctx.arc(x, y, 1 + (i % 3 === 0 ? 1 : 0), 0, Math.PI * 2);
    ctx.fill();
  }
  ctx.globalAlpha = 1;

  const padding = 42;
  const usableWidth = width - padding * 2;
  const usableHeight = height - padding * 2;

  const project = (position = {}) => ({
    x: padding + ((Number(position.x || 0) + arenaWidth / 2) / arenaWidth) * usableWidth,
    y: padding + ((arenaHeight / 2 - Number(position.y || 0)) / arenaHeight) * usableHeight,
  });

  // Arena perimeter boundaries
  ctx.strokeStyle = 'rgba(74, 144, 226, 0.35)';
  ctx.lineWidth = 2;
  ctx.setLineDash([8, 10]);
  ctx.strokeRect(padding, padding, width - padding * 2, height - padding * 2);
  ctx.setLineDash([]);

  // Bullets / plasma projectiles
  const bullets = snapshot.bullets || snapshot.projectiles || [];
  bullets.forEach((bullet) => {
    const point = project(bullet.position || bullet);
    ctx.fillStyle = '#ff7a45';
    ctx.beginPath();
    ctx.arc(point.x, point.y, 4, 0, Math.PI * 2);
    ctx.fill();

    ctx.strokeStyle = 'rgba(255, 122, 69, 0.4)';
    ctx.lineWidth = 6;
    ctx.stroke();
  });

  // Ships / fighters
  const fighters = snapshot.fighters || snapshot.entities || [];
  fighters.forEach((fighter, index) => {
    const point = project(fighter.position || fighter);
    const rotation = Number(fighter.rotation ?? fighter.rot ?? 0);
    const color = PLAYER_COLORS[index % PLAYER_COLORS.length];
    const isShieldActive = Boolean(fighter.shieldActive ?? fighter.shield);

    // Energy Shield effect
    if (isShieldActive) {
      ctx.strokeStyle = '#00f0ff';
      ctx.fillStyle = 'rgba(0, 240, 255, 0.12)';
      ctx.lineWidth = 3;
      ctx.beginPath();
      ctx.arc(point.x, point.y, 30, 0, Math.PI * 2);
      ctx.fill();
      ctx.stroke();
    }

    // Ship hull
    ctx.save();
    ctx.translate(point.x, point.y);
    ctx.rotate(rotation);
    ctx.fillStyle = color;
    ctx.strokeStyle = '#ffffff';
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.moveTo(25, 0);
    ctx.lineTo(-16, -13);
    ctx.lineTo(-9, 0);
    ctx.lineTo(-16, 13);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();
    ctx.restore();

    // Health bar
    const health = Math.max(0, Math.min(100, Number(fighter.health ?? fighter.hp ?? 0)));
    ctx.fillStyle = 'rgba(255, 255, 255, 0.2)';
    ctx.fillRect(point.x - 25, point.y + 34, 50, 5);
    ctx.fillStyle = health > 35 ? '#52c41a' : '#f5222d';
    ctx.fillRect(point.x - 25, point.y + 34, 50 * (health / 100), 5);

    // Pilot identifier label
    ctx.fillStyle = '#e6f7ff';
    ctx.font = '600 11px Inter, system-ui, sans-serif';
    ctx.textAlign = 'center';
    const label = String(fighter.playerId || fighter.id || `P${index + 1}`).slice(0, 12);
    ctx.fillText(label, point.x, point.y - 35);
  });
}
