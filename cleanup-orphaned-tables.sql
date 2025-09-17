-- QuestDB Cleanup Script for Orphaned Tables
-- Run this in QuestDB console to clean up tables created by session IDs

-- Show current tables to see what we're working with
SHOW TABLES;

-- Drop the orphaned tables (adjust table names as needed based on SHOW TABLES output)
-- These appear to be session IDs that were incorrectly used as table names

DROP TABLE IF EXISTS 7563905;
DROP TABLE IF EXISTS 8;

-- Keep only TelemetryTicks table
-- DROP TABLE IF EXISTS TelemetryTicks_backup_20250915_123456; -- Add any backup tables found

-- Verify cleanup
SHOW TABLES;

-- Check that TelemetryTicks has data
SELECT count(*) as total_records FROM TelemetryTicks;