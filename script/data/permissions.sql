-- Permissions seed
INSERT INTO permissions (id, description, active) VALUES
('read', 'Read resources and statistics', TRUE),
('write', 'Create and modify personal resources', TRUE),
('admin', 'Full administrative system access', TRUE),
('execute-match', 'Trigger and schedule match evaluations', TRUE),
('submit-agent', 'Upload and manage bot submissions', TRUE)
ON DUPLICATE KEY UPDATE description = VALUES(description);
