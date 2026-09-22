-- Seed: user_roles
-- Assigns roles to default users in the many-to-many RBAC table.

INSERT INTO user_roles (user_id, role_id)
SELECT id, 'admin' FROM users WHERE username = 'admin'
UNION ALL
SELECT id, 'player' FROM users WHERE username = 'pilot_alpha'
UNION ALL
SELECT id, 'player' FROM users WHERE username = 'pilot_beta'
ON CONFLICT (user_id, role_id) DO NOTHING;
