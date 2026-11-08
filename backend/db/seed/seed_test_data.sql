-- =============================================================================
-- Kladovki — TEST DATA (one separate test ЖК)
-- =============================================================================
-- How to run: Supabase → SQL Editor → paste this file → Run.
--   • Needs the app's tables: start the backend once against the database first
--     (it creates/updates the schema automatically).
--   • Safe to run more than once (existing rows are skipped).
--   • Everything lives in its own ЖК «ЖК Тестовый (seed)»; real ЖК, users and
--     payments are not touched. Every row id starts with "seed-".
--   • Remove it all with cleanup_test_data.sql.
--
-- Logins (password for everyone: test1234)
--   ОСИ admin ............ +77009990000   (phone)
--   Ержан  — active, 4 months left ........ ИИН 990101300011
--   Алия   — rental ends in 3 days ........ ИИН 990101400022
--   Болат  — overdue 10 days .............. ИИН 990101300033
--   Дана   — overdue 75 days (> 2 months) . ИИН 990101400044
--   Айгерим (rep. of ТОО «Тест Аптека»), commercial premises ... ИИН 990101400055
--   Самат  — asked for early termination .. ИИН 990101300066
--   Марат  — rental ended: released for debt (termination record) ИИН 990101300077
--   Тимур  — waiting for ОСИ approval ..... ИИН 990101300088
--   Сауле  — rejected by ОСИ .............. ИИН 990101400099
--   Нурлан — blocked by ОСИ ............... ИИН 990101300100
--
-- QR / registration link for this ЖК: <app>/login?jk=seed-jk
-- Note: the ИИН values are test numbers in an unused range (no real checksum).
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------- ОСИ + ЖК
-- The ЖК needs its owner and the owner belongs to the ЖК: user → ЖК → link.
INSERT INTO users (id, role, full_name, phone, email, password_hash, status)
VALUES ('seed-osi', 'admin', 'Тест ОСИ (seed)', '+77009990000', 'seed-osi@example.kz',
        '$2a$10$fPWXhsJX.7CU7ne5.vWnkO88XSRculAj2OMok.HGnCmEXAvb/0/gq', 'active')
ON CONFLICT DO NOTHING;

INSERT INTO jks (id, name, address, owner_id)
VALUES ('seed-jk', 'ЖК Тестовый (seed)', 'Астана, ул. Тестовая, 1', 'seed-osi')
ON CONFLICT DO NOTHING;

UPDATE users SET jk_id = 'seed-jk' WHERE id = 'seed-osi';

-- Tariff: 1 500 ₸ per m² per month (a 5 m² room costs 7 500 ₸/month).
INSERT INTO tariffs (id, jk_id, amount) VALUES ('seed-tariff', 'seed-jk', 1500) ON CONFLICT DO NOTHING;
INSERT INTO tariff_history (id, jk_id, amount, changed_at)
VALUES ('seed-tariff-h1', 'seed-jk', 1500, now() - interval '200 days') ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------- Residents
INSERT INTO users (id, role, full_name, phone, password_hash, jk_id, iin, status, review_comment,
                   owner_type, premises_kind, premises_number, company_bin, company_name)
VALUES
  ('seed-u01', 'resident', 'Ержан Активный',   '+77009990001', '$2a$10$fPWXhsJX.7CU7ne5.vWnkO88XSRculAj2OMok.HGnCmEXAvb/0/gq', 'seed-jk', '990101300011', 'active',   '', 'person',  'flat',       '12',  '', ''),
  ('seed-u02', 'resident', 'Алия Скоро-Конец', '+77009990002', '$2a$10$fPWXhsJX.7CU7ne5.vWnkO88XSRculAj2OMok.HGnCmEXAvb/0/gq', 'seed-jk', '990101400022', 'active',   '', 'person',  'flat',       '27',  '', ''),
  ('seed-u03', 'resident', 'Болат Просрочкин', '+77009990003', '$2a$10$fPWXhsJX.7CU7ne5.vWnkO88XSRculAj2OMok.HGnCmEXAvb/0/gq', 'seed-jk', '990101300033', 'active',   '', 'person',  'flat',       '45',  '', ''),
  ('seed-u04', 'resident', 'Дана Должникова',  '+77009990004', '$2a$10$fPWXhsJX.7CU7ne5.vWnkO88XSRculAj2OMok.HGnCmEXAvb/0/gq', 'seed-jk', '990101400044', 'active',   '', 'person',  'flat',       '58',  '', ''),
  ('seed-u05', 'resident', 'Айгерим Аптекова', '+77009990005', '$2a$10$fPWXhsJX.7CU7ne5.vWnkO88XSRculAj2OMok.HGnCmEXAvb/0/gq', 'seed-jk', '990101400055', 'active',   '', 'company', 'commercial', '3',   '990940012345', 'ТОО «Тест Аптека»'),
  ('seed-u06', 'resident', 'Самат Уезжающий',  '+77009990006', '$2a$10$fPWXhsJX.7CU7ne5.vWnkO88XSRculAj2OMok.HGnCmEXAvb/0/gq', 'seed-jk', '990101300066', 'active',   '', 'person',  'flat',       '71',  '', ''),
  ('seed-u07', 'resident', 'Марат Выселенный', '+77009990007', '$2a$10$fPWXhsJX.7CU7ne5.vWnkO88XSRculAj2OMok.HGnCmEXAvb/0/gq', 'seed-jk', '990101300077', 'active',   '', 'person',  'flat',       '84',  '', ''),
  ('seed-u08', 'resident', 'Тимур Новичок',    '+77009990008', '$2a$10$fPWXhsJX.7CU7ne5.vWnkO88XSRculAj2OMok.HGnCmEXAvb/0/gq', 'seed-jk', '990101300088', 'pending',  '', 'person',  'flat',       '96',  '', ''),
  ('seed-u09', 'resident', 'Сауле Отклонённая','+77009990009', '$2a$10$fPWXhsJX.7CU7ne5.vWnkO88XSRculAj2OMok.HGnCmEXAvb/0/gq', 'seed-jk', '990101400099', 'rejected', 'В доме нет квартиры 999 — проверьте номер', 'person', 'flat', '999', '', ''),
  ('seed-u10', 'resident', 'Нурлан Заблокир.', '+77009990010', '$2a$10$fPWXhsJX.7CU7ne5.vWnkO88XSRculAj2OMok.HGnCmEXAvb/0/gq', 'seed-jk', '990101300100', 'blocked',  'Квартира продана', 'person', 'flat', '101', '', '')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------- Storages
-- Status must match the rentals below: occupied = has an active rental.
INSERT INTO storages (id, jk_id, number, area, floor, entrance, status) VALUES
  ('seed-s01', 'seed-jk', 'Т-1',  5.0, -1, 1, 'occupied'),  -- Ержан
  ('seed-s02', 'seed-jk', 'Т-2',  4.0, -1, 1, 'occupied'),  -- Алия
  ('seed-s03', 'seed-jk', 'Т-3',  3.5, -1, 1, 'occupied'),  -- Болат (overdue)
  ('seed-s04', 'seed-jk', 'Т-4',  6.0, -1, 2, 'occupied'),  -- Дана (overdue > 2 months)
  ('seed-s05', 'seed-jk', 'Т-5',  8.0, -1, 2, 'occupied'),  -- ТОО «Тест Аптека»
  ('seed-s06', 'seed-jk', 'Т-6',  4.5, -1, 2, 'occupied'),  -- Самат (termination requested)
  ('seed-s07', 'seed-jk', 'Т-7',  3.0, -2, 1, 'free'),      -- Марат's old room, released
  ('seed-s08', 'seed-jk', 'Т-8',  2.5, -2, 1, 'free'),
  ('seed-s09', 'seed-jk', 'Т-9',  4.0, -2, 2, 'free'),
  ('seed-s10', 'seed-jk', 'Т-10', 5.5, -2, 2, 'free'),
  ('seed-s11', 'seed-jk', 'Т-11', 3.0,  0, 3, 'free'),
  ('seed-s12', 'seed-jk', 'Т-12', 7.0,  0, 3, 'free')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------- Rentals
-- price_per_month = round(area × 1 500); total_paid = what was paid.
INSERT INTO rentals (id, storage_id, user_id, jk_id, months, price_per_month, total_paid, starts_at, ends_at, status, created_at) VALUES
  -- Ержан: 6 months (booking 3 + renewal 3), 4 months left
  ('seed-r01', 'seed-s01', 'seed-u01', 'seed-jk', 6,  7500, 45000, now() - interval '60 days',  now() + interval '122 days', 'active', now() - interval '60 days'),
  -- Алия: ends in 3 days → «Срок аренды истекает»
  ('seed-r02', 'seed-s02', 'seed-u02', 'seed-jk', 1,  6000,  6000, now() - interval '27 days',  now() + interval '3 days',   'active', now() - interval '27 days'),
  -- Болат: ended 10 days ago, still occupies the room → debt
  ('seed-r03', 'seed-s03', 'seed-u03', 'seed-jk', 1,  5250,  5250, now() - interval '40 days',  now() - interval '10 days',  'active', now() - interval '40 days'),
  -- Дана: ended 75 days ago (> 2 months) → debt warning, ОСИ may terminate
  ('seed-r04', 'seed-s04', 'seed-u04', 'seed-jk', 2,  9000, 18000, now() - interval '136 days', now() - interval '75 days',  'active', now() - interval '136 days'),
  -- ТОО «Тест Аптека»: 12 months, paid up
  ('seed-r05', 'seed-s05', 'seed-u05', 'seed-jk', 12, 12000, 144000, now() - interval '100 days', now() + interval '265 days', 'active', now() - interval '100 days'),
  -- Самат: 3 months, asked to end early
  ('seed-r06', 'seed-s06', 'seed-u06', 'seed-jk', 3,  6750, 20250, now() - interval '30 days',  now() + interval '61 days',  'active', now() - interval '30 days'),
  -- Марат: overdue and released by the ОСИ for debt (see terminations)
  ('seed-r07', 'seed-s07', 'seed-u07', 'seed-jk', 1,  4500,  4500, now() - interval '150 days', now() - interval '80 days',  'expired', now() - interval '150 days')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------- Payments (Kaspi)
INSERT INTO payments (id, rental_id, user_id, amount, provider, status, kind, months, created_at) VALUES
  ('seed-p01',  'seed-r01', 'seed-u01', 22500,  'apipay', 'success', 'booking', 0, now() - interval '60 days'),
  ('seed-p01b', 'seed-r01', 'seed-u01', 22500,  'apipay', 'success', 'renewal', 3, now() - interval '5 days'),
  ('seed-p02',  'seed-r02', 'seed-u02', 6000,   'apipay', 'success', 'booking', 0, now() - interval '27 days'),
  ('seed-p03',  'seed-r03', 'seed-u03', 5250,   'apipay', 'success', 'booking', 0, now() - interval '40 days'),
  ('seed-p04',  'seed-r04', 'seed-u04', 18000,  'apipay', 'success', 'booking', 0, now() - interval '136 days'),
  ('seed-p05',  'seed-r05', 'seed-u05', 144000, 'apipay', 'success', 'booking', 0, now() - interval '100 days'),
  ('seed-p06',  'seed-r06', 'seed-u06', 20250,  'apipay', 'success', 'booking', 0, now() - interval '30 days'),
  ('seed-p07',  'seed-r07', 'seed-u07', 4500,   'apipay', 'success', 'booking', 0, now() - interval '150 days'),
  -- a declined Kaspi invoice (shows as «Ошибка» in Платежи)
  ('seed-p08',  'seed-r02', 'seed-u02', 6000,   'apipay', 'failed',  'renewal', 1, now() - interval '1 day')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------- Requests (Заявки)
-- Самат's early termination: ~2/3 of 20 250 ₸ unused → about 13 500 ₸ to return.
INSERT INTO requests (id, jk_id, user_id, rental_id, type, status, amount, comment, created_at) VALUES
  ('seed-q01', 'seed-jk', 'seed-u06', 'seed-r06', 'early_termination', 'open', 13500,
   'Переезжаю в другой город, прошу вернуть остаток', now() - interval '1 day')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------- Termination record (п. 5.2)
INSERT INTO terminations (id, rental_id, reason, note, police, costs, created_by, created_at) VALUES
  ('seed-t01', 'seed-r07', 'Задолженность более 2 месяцев',
   'Вскрыто в присутствии председателя ОСИ. Опись: 2 коробки, велосипед. Вещи хранятся в помещении ОСИ.',
   true, 10000, 'seed-osi', now() - interval '3 days')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------- Agreements accepted
-- Residents accepted the user agreement at registration; the ОСИ accepted the
-- regulation, so the admin panel opens without the prompt.
INSERT INTO agreement_acceptances (id, user_id, document, version, context)
SELECT 'seed-acc-' || u.id, u.id,
       CASE WHEN u.role = 'admin' THEN 'osi_rules' ELSE 'resident_terms' END,
       '2026-09-21',
       CASE WHEN u.role = 'admin' THEN 'prompt' ELSE 'registration' END
FROM users u WHERE u.id LIKE 'seed-%'
ON CONFLICT DO NOTHING;

COMMIT;

-- Reminders (ending soon, overdue, debt > 2 months) are generated by the
-- server's hourly job — they appear in «Уведомления» within an hour, or right
-- after the next server restart.
