-- Agentrix database user
DROP USER IF EXISTS 'agentrix'@'localhost';
CREATE USER 'agentrix'@'localhost' IDENTIFIED BY 'agentrix';
GRANT ALL PRIVILEGES ON agentrix.* TO 'agentrix'@'localhost';
GRANT ALL PRIVILEGES ON capsule.* TO 'agentrix'@'localhost';
FLUSH PRIVILEGES;
