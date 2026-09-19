# Starfighter reference bots

Three reference strategies, each a plain Python 3 script speaking the
Agentrix `init` / `perception` / `action` / `end` JSON Lines protocol over
stdin/stdout. They use only the Python standard library.

- `bot_random.py` -- uniformly random action every tick. Comparison floor.
- `bot_hunter.py` -- aggressive: closes distance, turns toward and shoots
  the nearest rival in radar range.
- `bot_evasive.py` -- defensive: flees and raises shield when a rival
  gets within its danger range, coasts and regenerates energy otherwise.

The Starfighter manifest selects `bot_hunter.py` or `bot_evasive.py` when a
scheduled match lacks one or both participant submissions. The worker keeps
each script alive for the complete match and terminates it immediately if a
tick response exceeds the configured timeout.
