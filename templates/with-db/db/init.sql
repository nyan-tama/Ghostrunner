-- Initial setup (table creation is handled by GORM AutoMigrate)
-- This file runs on first docker-compose up

-- Create extension if needed
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
