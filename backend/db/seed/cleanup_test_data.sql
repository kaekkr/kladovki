-- =============================================================================
-- Kladovki — remove the TEST DATA created by seed_test_data.sql
-- =============================================================================
-- Deletes the test ЖК «ЖК Тестовый (seed)» (id seed-jk) and everything in it:
-- its storages, rentals, payments, requests, termination records, tariff, and
-- all users of that ЖК (the seed-* users plus anyone who registered into the
-- test ЖК later via its link). Nothing outside the test ЖК is touched.
--
-- How to run: Supabase → SQL Editor → paste → Run.
-- =============================================================================

BEGIN;

-- Users to remove: the seed users and anyone who joined the test ЖК.
CREATE TEMP TABLE seed_users ON COMMIT DROP AS
  SELECT id FROM users WHERE id LIKE 'seed-%' OR jk_id = 'seed-jk';

DELETE FROM terminations WHERE rental_id IN (SELECT id FROM rentals WHERE jk_id = 'seed-jk');
DELETE FROM requests     WHERE jk_id = 'seed-jk';
DELETE FROM payments     WHERE rental_id IN (SELECT id FROM rentals WHERE jk_id = 'seed-jk');
DELETE FROM rentals      WHERE jk_id = 'seed-jk';
DELETE FROM storages     WHERE jk_id = 'seed-jk';
DELETE FROM tariff_history WHERE jk_id = 'seed-jk';
DELETE FROM tariffs      WHERE jk_id = 'seed-jk';

-- Per-user rows (most also cascade, listed for clarity).
DELETE FROM notification_deliveries
 WHERE notification_id IN (SELECT id FROM notifications WHERE user_id IN (SELECT id FROM seed_users));
DELETE FROM notifications         WHERE user_id IN (SELECT id FROM seed_users);
DELETE FROM agreement_acceptances WHERE user_id IN (SELECT id FROM seed_users);
DELETE FROM telegram_link_tokens  WHERE user_id IN (SELECT id FROM seed_users);

-- The ЖК points to its ОСИ and users point to the ЖК: unlink, then delete.
UPDATE users SET jk_id = NULL WHERE id IN (SELECT id FROM seed_users);
DELETE FROM jks   WHERE id = 'seed-jk';
DELETE FROM users WHERE id IN (SELECT id FROM seed_users);

COMMIT;
