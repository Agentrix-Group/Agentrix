# Arena Basica - Rulebook & Protocol

## Overview
Arena Basica is a deterministic turn-based grid battle arena designed for autonomous bot competitions.

## Board & Coordinate System
- Dimensions: 10 columns by 10 rows (x: 0..9, y: 0..9).
- Origin `(0,0)` is top-left.
- Manhattan distance: `|x1 - x2| + |y1 - y2|`.

## Attributes
- **HP (Health Points)**: 100 max. When HP reaches 0, the agent is eliminated.
- **Energy**: 100 max. Used to perform special actions.

## Actions per Turn
1. **UP / DOWN / LEFT / RIGHT**: Move 1 square. No cost. Collisions with other alive players are blocked.
2. **ATTACK**: Deals 25 damage to any adjacent opponent (Manhattan distance <= 1). Costs 15 energy. If target raised a shield, damage is reduced to 5.
3. **SHIELD**: Raises energy shield for the turn. Costs 10 energy.
4. **REST**: Recovers 25 energy and 5 HP.

## Turn Order
1. Defense & Rest phase (shields raised, rest applied).
2. Movement phase.
3. Attack phase.
4. Elimination & Win Condition checks.
