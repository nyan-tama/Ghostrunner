-- Sample data (table creation is handled by GORM AutoMigrate)
-- This file runs on first docker-compose up

INSERT INTO samples (name, description, created_at, updated_at) VALUES
  ('Sample Item 1', 'This is the first sample item', NOW(), NOW()),
  ('Sample Item 2', 'This is the second sample item', NOW(), NOW()),
  ('Sample Item 3', 'This is the third sample item', NOW(), NOW());
