-- ============================================================
-- mechatracker — Migration 003: Dev test data
-- F-150 engine repairs + 1989 Jeep Cherokee suspension
-- Dev org: 00000000-0000-0000-0000-000000000003
-- Dev user: 00000000-0000-0000-0000-000000000002
-- ============================================================

-- ── Vehicles ─────────────────────────────────────────────────────────────────

-- 2019 Ford F-150 (insert if missing, update mileage if already present)
INSERT INTO vehicles (id, org_id, year, make, model, trim, mileage)
VALUES (
    'a44d8522-66b3-4146-af64-9db60c72470a',
    '00000000-0000-0000-0000-000000000003',
    2019,
    'Ford',
    'F-150',
    'XLT',
    84200
) ON CONFLICT (id) DO UPDATE SET mileage = 84200;

-- 1989 Jeep Cherokee (XJ)
INSERT INTO vehicles (id, org_id, year, make, model, trim, mileage)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000003',
    1989,
    'Jeep',
    'Cherokee',
    'XJ Sport',
    187400
) ON CONFLICT DO NOTHING;


-- ── F-150: Spark plug replacement (completed) ─────────────────────────────────

INSERT INTO repairs (id, vehicle_id, org_id, created_by, title, system, status, progress_pct, total_parts_cost, started_at, completed_at)
VALUES (
    'aaaaaaaa-0001-0001-0001-000000000001',
    'a44d8522-66b3-4146-af64-9db60c72470a',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    'Spark plug replacement',
    'engine',
    'completed',
    100,
    72.00,
    NOW() - INTERVAL '45 days',
    NOW() - INTERVAL '44 days'
) ON CONFLICT DO NOTHING;

INSERT INTO repair_steps (repair_id, position, title, notes, torque_spec, completed, completed_at) VALUES
    ('aaaaaaaa-0001-0001-0001-000000000001', 1, 'Let engine cool completely',        'At least 30 min after running — burns are no fun',          NULL,        true, NOW() - INTERVAL '44 days'),
    ('aaaaaaaa-0001-0001-0001-000000000001', 2, 'Remove engine cover',               '4 push-clips, pull straight up',                            NULL,        true, NOW() - INTERVAL '44 days'),
    ('aaaaaaaa-0001-0001-0001-000000000001', 3, 'Disconnect coil-on-plug connectors','Press tab, wiggle out — do one at a time to avoid mixing',  NULL,        true, NOW() - INTERVAL '44 days'),
    ('aaaaaaaa-0001-0001-0001-000000000001', 4, 'Remove COP bolts and pull coils',   '8mm bolt per coil',                                         NULL,        true, NOW() - INTERVAL '44 days'),
    ('aaaaaaaa-0001-0001-0001-000000000001', 5, 'Remove old plugs',                  'Use 9/16" spark plug socket with extension',                NULL,        true, NOW() - INTERVAL '44 days'),
    ('aaaaaaaa-0001-0001-0001-000000000001', 6, 'Install new plugs',                 'Thread by hand first — EcoBoost head is aluminum, easy to cross-thread', '13 ft-lb', true, NOW() - INTERVAL '44 days'),
    ('aaaaaaaa-0001-0001-0001-000000000001', 7, 'Reinstall coils and reconnect',     'Torque COP bolts snug',                                     '62 in-lb',  true, NOW() - INTERVAL '44 days'),
    ('aaaaaaaa-0001-0001-0001-000000000001', 8, 'Start engine, check for misfires',  'Clear any stored codes with OBD2 reader',                   NULL,        true, NOW() - INTERVAL '44 days')
ON CONFLICT DO NOTHING;

INSERT INTO parts (repair_id, name, part_number, supplier, status, cost) VALUES
    ('aaaaaaaa-0001-0001-0001-000000000001', 'Motorcraft SP-537 spark plugs (x6)', 'SP-537',    'Ford dealer', 'installed', 72.00)
ON CONFLICT DO NOTHING;

INSERT INTO repair_notes (repair_id, user_id, body) VALUES
    ('aaaaaaaa-0001-0001-0001-000000000001', '00000000-0000-0000-0000-000000000002',
     'Plugs looked rough — one had a hairline crack on the porcelain. Glad I caught it at 60k. Running noticeably smoother after.')
ON CONFLICT DO NOTHING;

INSERT INTO repair_history (
    repair_id, vehicle_id, org_id, created_by,
    vehicle_snapshot, title, system,
    total_parts_cost, total_steps,
    steps_snapshot, parts_snapshot, notes_snapshot, resources_snapshot,
    started_at, completed_at
) VALUES (
    'aaaaaaaa-0001-0001-0001-000000000001',
    'a44d8522-66b3-4146-af64-9db60c72470a',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    '2019 Ford F-150 XLT',
    'Spark plug replacement',
    'engine',
    72.00, 8,
    '[]'::jsonb, '[]'::jsonb, '[]'::jsonb, '[]'::jsonb,
    NOW() - INTERVAL '45 days',
    NOW() - INTERVAL '44 days'
) ON CONFLICT DO NOTHING;


-- ── F-150: Oil & filter change (completed) ────────────────────────────────────

INSERT INTO repairs (id, vehicle_id, org_id, created_by, title, system, status, progress_pct, total_parts_cost, started_at, completed_at)
VALUES (
    'aaaaaaaa-0002-0002-0002-000000000002',
    'a44d8522-66b3-4146-af64-9db60c72470a',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    'Oil & filter change',
    'engine',
    'completed',
    100,
    44.00,
    NOW() - INTERVAL '14 days',
    NOW() - INTERVAL '14 days'
) ON CONFLICT DO NOTHING;

INSERT INTO repair_steps (repair_id, position, title, notes, torque_spec, completed, completed_at) VALUES
    ('aaaaaaaa-0002-0002-0002-000000000002', 1, 'Warm engine 5 min',          'Warm oil drains faster and carries more sludge out',         NULL,        true, NOW() - INTERVAL '14 days'),
    ('aaaaaaaa-0002-0002-0002-000000000002', 2, 'Position drain pan',         'Oil comes out fast at first — keep hands clear',             NULL,        true, NOW() - INTERVAL '14 days'),
    ('aaaaaaaa-0002-0002-0002-000000000002', 3, 'Remove drain plug',          '15mm socket',                                                NULL,        true, NOW() - INTERVAL '14 days'),
    ('aaaaaaaa-0002-0002-0002-000000000002', 4, 'Remove and replace filter',  'Lightly oil the new gasket before installing',               NULL,        true, NOW() - INTERVAL '14 days'),
    ('aaaaaaaa-0002-0002-0002-000000000002', 5, 'Reinstall drain plug',       'New crush washer every time',                                '26 ft-lb',  true, NOW() - INTERVAL '14 days'),
    ('aaaaaaaa-0002-0002-0002-000000000002', 6, 'Fill with 6 qt 5W-30',       'Add 5.5 qt, run briefly, check dipstick and top off',        NULL,        true, NOW() - INTERVAL '14 days'),
    ('aaaaaaaa-0002-0002-0002-000000000002', 7, 'Reset oil life monitor',     'Hold OK in Settings > Vehicle > Oil Life > Reset',           NULL,        true, NOW() - INTERVAL '14 days')
ON CONFLICT DO NOTHING;

INSERT INTO parts (repair_id, name, part_number, supplier, status, cost) VALUES
    ('aaaaaaaa-0002-0002-0002-000000000002', 'Motorcraft FL-500-S oil filter', 'FL-500-S', 'AutoZone', 'installed', 8.00),
    ('aaaaaaaa-0002-0002-0002-000000000002', 'Motorcraft 5W-30 full synthetic (6 qt)', NULL, 'AutoZone', 'installed', 34.00),
    ('aaaaaaaa-0002-0002-0002-000000000002', 'Drain plug crush washer', NULL, 'AutoZone', 'installed', 2.00)
ON CONFLICT DO NOTHING;

INSERT INTO repair_notes (repair_id, user_id, body) VALUES
    ('aaaaaaaa-0002-0002-0002-000000000002', '00000000-0000-0000-0000-000000000002',
     'Quick and clean. Oil looked good, no metal flakes. Due again at ~90k.')
ON CONFLICT DO NOTHING;

INSERT INTO repair_history (
    repair_id, vehicle_id, org_id, created_by,
    vehicle_snapshot, title, system,
    total_parts_cost, total_steps,
    steps_snapshot, parts_snapshot, notes_snapshot, resources_snapshot,
    started_at, completed_at
) VALUES (
    'aaaaaaaa-0002-0002-0002-000000000002',
    'a44d8522-66b3-4146-af64-9db60c72470a',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    '2019 Ford F-150 XLT',
    'Oil & filter change',
    'engine',
    44.00, 7,
    '[]'::jsonb, '[]'::jsonb, '[]'::jsonb, '[]'::jsonb,
    NOW() - INTERVAL '14 days',
    NOW() - INTERVAL '14 days'
) ON CONFLICT DO NOTHING;


-- ── F-150: Fuel injector cleaning (in progress) ───────────────────────────────

INSERT INTO repairs (id, vehicle_id, org_id, created_by, title, system, status, progress_pct, total_parts_cost, started_at)
VALUES (
    'aaaaaaaa-0003-0003-0003-000000000003',
    'a44d8522-66b3-4146-af64-9db60c72470a',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    'Fuel injector cleaning',
    'engine',
    'in_progress',
    33,
    18.00,
    NOW() - INTERVAL '2 days'
) ON CONFLICT DO NOTHING;

INSERT INTO repair_steps (repair_id, position, title, notes, torque_spec, completed, completed_at) VALUES
    ('aaaaaaaa-0003-0003-0003-000000000003', 1, 'Add BG 44K to fuel tank',       'Pour full bottle into tank before filling up — needs to run through a full tank', NULL, true,  NOW() - INTERVAL '2 days'),
    ('aaaaaaaa-0003-0003-0003-000000000003', 2, 'Run tank down to near empty',   'Let the treatment work through the system',                                        NULL, false, NULL),
    ('aaaaaaaa-0003-0003-0003-000000000003', 3, 'Check for improved idle quality','Should smooth out — if not, may need bench cleaning',                             NULL, false, NULL)
ON CONFLICT DO NOTHING;

INSERT INTO parts (repair_id, name, part_number, supplier, status, cost) VALUES
    ('aaaaaaaa-0003-0003-0003-000000000003', 'BG 44K fuel system cleaner', '44K', 'AutoZone', 'installed', 18.00)
ON CONFLICT DO NOTHING;

INSERT INTO repair_notes (repair_id, user_id, body) VALUES
    ('aaaaaaaa-0003-0003-0003-000000000003', '00000000-0000-0000-0000-000000000002',
     'Noticed slight rough idle at cold start, more noticeable in the last month. Starting with in-tank treatment before pulling injectors.')
ON CONFLICT DO NOTHING;


-- ── Jeep: Front axle u-joints (completed) ─────────────────────────────────────

INSERT INTO repairs (id, vehicle_id, org_id, created_by, title, system, status, progress_pct, total_parts_cost, started_at, completed_at)
VALUES (
    'bbbbbbbb-0001-0001-0001-000000000001',
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    'Front axle u-joint replacement',
    'suspension',
    'completed',
    100,
    68.00,
    NOW() - INTERVAL '60 days',
    NOW() - INTERVAL '58 days'
) ON CONFLICT DO NOTHING;

INSERT INTO repair_steps (repair_id, position, title, notes, torque_spec, completed, completed_at) VALUES
    ('bbbbbbbb-0001-0001-0001-000000000001', 1, 'Raise and support front axle',       'Stands under the frame — not the axle',                               NULL,        true, NOW() - INTERVAL '58 days'),
    ('bbbbbbbb-0001-0001-0001-000000000001', 2, 'Remove front wheels',                '6-lug, 13/16" socket',                                                NULL,        true, NOW() - INTERVAL '58 days'),
    ('bbbbbbbb-0001-0001-0001-000000000001', 3, 'Remove brake caliper and rotor',     'Hang caliper — do not let it dangle by the line',                     NULL,        true, NOW() - INTERVAL '58 days'),
    ('bbbbbbbb-0001-0001-0001-000000000001', 4, 'Remove axle shaft',                  'Disconnect from diff and pull through knuckle',                       NULL,        true, NOW() - INTERVAL '58 days'),
    ('bbbbbbbb-0001-0001-0001-000000000001', 5, 'Press out old u-joints',             'U-joint press or vise with sockets — do not hammer if you can avoid it', NULL,      true, NOW() - INTERVAL '58 days'),
    ('bbbbbbbb-0001-0001-0001-000000000001', 6, 'Press in new u-joints',              'Grease zerks facing accessible direction',                            NULL,        true, NOW() - INTERVAL '58 days'),
    ('bbbbbbbb-0001-0001-0001-000000000001', 7, 'Reinstall axle, rotor, caliper',     NULL,                                                                  '90 ft-lb',  true, NOW() - INTERVAL '58 days'),
    ('bbbbbbbb-0001-0001-0001-000000000001', 8, 'Reinstall wheels and torque lugs',   NULL,                                                                  '95 ft-lb',  true, NOW() - INTERVAL '58 days'),
    ('bbbbbbbb-0001-0001-0001-000000000001', 9, 'Grease new u-joints',                'Pump grease until it purges from all four caps',                      NULL,        true, NOW() - INTERVAL '58 days')
ON CONFLICT DO NOTHING;

INSERT INTO parts (repair_id, name, part_number, supplier, status, cost) VALUES
    ('bbbbbbbb-0001-0001-0001-000000000001', 'Spicer 5-760X u-joints (x2)', '5-760X',  'RockAuto', 'installed', 48.00),
    ('bbbbbbbb-0001-0001-0001-000000000001', 'Grease (cartridge)',           NULL,       'AutoZone', 'installed', 8.00),
    ('bbbbbbbb-0001-0001-0001-000000000001', 'Snap ring set',                NULL,       'AutoZone', 'installed', 6.00),
    ('bbbbbbbb-0001-0001-0001-000000000001', 'Shop rags',                    NULL,       'shop',     'installed', 6.00)
ON CONFLICT DO NOTHING;

INSERT INTO repair_notes (repair_id, user_id, body) VALUES
    ('bbbbbbbb-0001-0001-0001-000000000001', '00000000-0000-0000-0000-000000000002',
     'Classic XJ clunk under acceleration — both front u-joints were trashed, factory units with no grease zerks. Replaced with Spicer greaseable units.'),
    ('bbbbbbbb-0001-0001-0001-000000000001', '00000000-0000-0000-0000-000000000002',
     'Clunk completely gone. Grease every 5k miles going forward.')
ON CONFLICT DO NOTHING;

INSERT INTO repair_history (
    repair_id, vehicle_id, org_id, created_by,
    vehicle_snapshot, title, system,
    total_parts_cost, total_steps,
    steps_snapshot, parts_snapshot, notes_snapshot, resources_snapshot,
    started_at, completed_at
) VALUES (
    'bbbbbbbb-0001-0001-0001-000000000001',
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    '1989 Jeep Cherokee XJ Sport',
    'Front axle u-joint replacement',
    'suspension',
    68.00, 9,
    '[]'::jsonb, '[]'::jsonb, '[]'::jsonb, '[]'::jsonb,
    NOW() - INTERVAL '60 days',
    NOW() - INTERVAL '58 days'
) ON CONFLICT DO NOTHING;


-- ── Jeep: Lift kit install (in progress) ──────────────────────────────────────

INSERT INTO repairs (id, vehicle_id, org_id, created_by, title, system, status, progress_pct, total_parts_cost, started_at)
VALUES (
    'bbbbbbbb-0002-0002-0002-000000000002',
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    '3" suspension lift kit install',
    'suspension',
    'pending_parts',
    20,
    487.00,
    NOW() - INTERVAL '5 days'
) ON CONFLICT DO NOTHING;

INSERT INTO repair_steps (repair_id, position, title, notes, torque_spec, completed, completed_at) VALUES
    ('bbbbbbbb-0002-0002-0002-000000000002', 1,  'Verify all parts received',           'Check kit contents against invoice — missing a track bar bracket last order', NULL,       true,  NOW() - INTERVAL '5 days'),
    ('bbbbbbbb-0002-0002-0002-000000000002', 2,  'Raise vehicle, support frame on stands', 'Front and rear — this is a full-day job',                               NULL,       false, NULL),
    ('bbbbbbbb-0002-0002-0002-000000000002', 3,  'Remove front coil springs',           'Compress springs before removing — they are under load',                   NULL,       false, NULL),
    ('bbbbbbbb-0002-0002-0002-000000000002', 4,  'Install new front coils and spacers', 'Rough Country 3" springs',                                                 NULL,       false, NULL),
    ('bbbbbbbb-0002-0002-0002-000000000002', 5,  'Install front track bar drop bracket','Required to re-center the axle — do not skip',                             NULL,       false, NULL),
    ('bbbbbbbb-0002-0002-0002-000000000002', 6,  'Replace front shocks',                'Extended travel shocks to match lift',                                      NULL,       false, NULL),
    ('bbbbbbbb-0002-0002-0002-000000000002', 7,  'Remove rear leaf springs',            NULL,                                                                        NULL,       false, NULL),
    ('bbbbbbbb-0002-0002-0002-000000000002', 8,  'Install add-a-leaf packs',            'Add-a-leaf for rear lift, preserves stock spring feel',                    NULL,       false, NULL),
    ('bbbbbbbb-0002-0002-0002-000000000002', 9,  'Replace rear shocks',                 NULL,                                                                        NULL,       false, NULL),
    ('bbbbbbbb-0002-0002-0002-000000000002', 10, 'Torque all suspension fasteners',     'Torque with vehicle at ride height, not hanging',                          'varies',   false, NULL),
    ('bbbbbbbb-0002-0002-0002-000000000002', 11, 'Alignment',                           'Must align after any suspension work — do not skip',                       NULL,       false, NULL)
ON CONFLICT DO NOTHING;

INSERT INTO parts (repair_id, name, part_number, supplier, status, cost) VALUES
    ('bbbbbbbb-0002-0002-0002-000000000002', 'Rough Country 3" lift kit (XJ)',   '636',   'Rough Country', 'in_stock',  319.00),
    ('bbbbbbbb-0002-0002-0002-000000000002', 'Front track bar drop bracket',     NULL,    'Rough Country', 'in_stock',  38.00),
    ('bbbbbbbb-0002-0002-0002-000000000002', 'Extended front shocks (x2)',       NULL,    'Rough Country', 'ordered',   72.00),
    ('bbbbbbbb-0002-0002-0002-000000000002', 'Extended rear shocks (x2)',        NULL,    'Rough Country', 'ordered',   58.00)
ON CONFLICT DO NOTHING;

INSERT INTO repair_notes (repair_id, user_id, body) VALUES
    ('bbbbbbbb-0002-0002-0002-000000000002', '00000000-0000-0000-0000-000000000002',
     'Going with Rough Country 3" — good budget option for a trail rig. Ordered the shocks separately since they were backordered in the kit.'),
    ('bbbbbbbb-0002-0002-0002-000000000002', '00000000-0000-0000-0000-000000000002',
     'Shocks shipped today, ETA 3 days. Will finish the install once they arrive.')
ON CONFLICT DO NOTHING;


-- ── Jeep: Sway bar end links (completed) ──────────────────────────────────────

INSERT INTO repairs (id, vehicle_id, org_id, created_by, title, system, status, progress_pct, total_parts_cost, started_at, completed_at)
VALUES (
    'bbbbbbbb-0003-0003-0003-000000000003',
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    'Front sway bar end link replacement',
    'suspension',
    'completed',
    100,
    28.00,
    NOW() - INTERVAL '90 days',
    NOW() - INTERVAL '90 days'
) ON CONFLICT DO NOTHING;

INSERT INTO repair_steps (repair_id, position, title, notes, torque_spec, completed, completed_at) VALUES
    ('bbbbbbbb-0003-0003-0003-000000000003', 1, 'Raise and support vehicle',  '2-post lift or stands under frame',   NULL,       true, NOW() - INTERVAL '90 days'),
    ('bbbbbbbb-0003-0003-0003-000000000003', 2, 'Remove old end links',       '15mm on both ends, hold stud with pliers if spinning', NULL, true, NOW() - INTERVAL '90 days'),
    ('bbbbbbbb-0003-0003-0003-000000000003', 3, 'Install new end links',      'Adjustable links — set to stock length until aligned',  '45 ft-lb', true, NOW() - INTERVAL '90 days'),
    ('bbbbbbbb-0003-0003-0003-000000000003', 4, 'Lower and check clearance',  'No binding through full steering lock-to-lock',         NULL,       true, NOW() - INTERVAL '90 days')
ON CONFLICT DO NOTHING;

INSERT INTO parts (repair_id, name, part_number, supplier, status, cost) VALUES
    ('bbbbbbbb-0003-0003-0003-000000000003', 'Moog adjustable end links (pair)', 'K750109', 'RockAuto', 'installed', 28.00)
ON CONFLICT DO NOTHING;

INSERT INTO repair_notes (repair_id, user_id, body) VALUES
    ('bbbbbbbb-0003-0003-0003-000000000003', '00000000-0000-0000-0000-000000000002',
     'Stock links were cracked — classic XJ failure at this mileage. Got adjustable Moog units so I can tune once the lift is in.')
ON CONFLICT DO NOTHING;

INSERT INTO repair_history (
    repair_id, vehicle_id, org_id, created_by,
    vehicle_snapshot, title, system,
    total_parts_cost, total_steps,
    steps_snapshot, parts_snapshot, notes_snapshot, resources_snapshot,
    started_at, completed_at
) VALUES (
    'bbbbbbbb-0003-0003-0003-000000000003',
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    '1989 Jeep Cherokee XJ Sport',
    'Front sway bar end link replacement',
    'suspension',
    28.00, 4,
    '[]'::jsonb, '[]'::jsonb, '[]'::jsonb, '[]'::jsonb,
    NOW() - INTERVAL '90 days',
    NOW() - INTERVAL '90 days'
) ON CONFLICT DO NOTHING;