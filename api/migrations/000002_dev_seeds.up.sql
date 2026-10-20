-- Only runs in local dev — never promoted to staging/prod migrations.
-- Seeded org and user match the UUIDs injected by DEV_AUTH_BYPASS=true
-- in middleware/auth.go (DevOrgID / DevUserID constants).

INSERT INTO organizations (id, name, slug, plan, personal)
VALUES (
    '00000000-0000-0000-0000-000000000003',
    'Dev Org',
    'dev-org',
    'personal',
    true
) ON CONFLICT DO NOTHING;

INSERT INTO users (id, google_sub, org_id, role, email, email_verified, name)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'dev_google_sub_local',
    '00000000-0000-0000-0000-000000000003',
    'owner',
    'dev@mechatracker.local',
    true,
    'Dev User'
) ON CONFLICT DO NOTHING;