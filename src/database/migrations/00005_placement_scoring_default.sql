-- +goose Up
-- +goose StatementBegin

-- ADR-0013: los concursos nuevos puntúan por posición final (Starfighter
-- 0.4.0, de 2 a 5 jugadores) con las bajas como primer desempate. Solo
-- cambia el valor por defecto: los concursos existentes conservan la
-- política guardada, que sin "mode" se interpreta como victoria/empate/
-- derrota, y no se reinterpretan sus resultados.
ALTER TABLE contests
ALTER COLUMN scoring_policy
SET DEFAULT '{"mode":"placement","win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0,"tiebreakers":["kills","score_diff","head_to_head","wins"]}'::jsonb;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE contests
ALTER COLUMN scoring_policy
SET DEFAULT '{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0,"tiebreakers":["score_diff","head_to_head","wins"]}'::jsonb;

-- +goose StatementEnd
