-- Seed: users
-- Default system accounts with SHA-512 hashed credentials.
-- Passwords:
--   admin: admin123
--   pilot_alpha: pilot123
--   pilot_beta: pilot123

INSERT INTO users (id, username, email, password, role_id, active) VALUES
('usr-admin-001', 'admin', 'admin@agentrix.local', '7fcf4ba391c48784edde599889d6e3f1e47a27db36ecc050cc92f259bfac38afad2c68a1ae804d77075e8fb722503f3eca2b2c1006ee6f6c7b7628cb45fffd1d', 'admin', TRUE),
('usr-pilot-001', 'pilot_alpha', 'pilot_alpha@agentrix.local', '5047300a58f3abe4458de0b0eb4316a1dcad86db8683ebcfb3ccbd096a28dd71c1324759698b3126c16e4b993fc5d84de58ecd937edc37c6bb8ab0cfefb57e27', 'player', TRUE),
('usr-pilot-002', 'pilot_beta', 'pilot_beta@agentrix.local', '5047300a58f3abe4458de0b0eb4316a1dcad86db8683ebcfb3ccbd096a28dd71c1324759698b3126c16e4b993fc5d84de58ecd937edc37c6bb8ab0cfefb57e27', 'player', TRUE)
ON CONFLICT (username) DO UPDATE SET 
    email = EXCLUDED.email,
    password = EXCLUDED.password,
    role_id = EXCLUDED.role_id, 
    active = EXCLUDED.active;
