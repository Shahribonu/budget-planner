-- Consent to the current terms and privacy policy, and export throttling.
alter table users
    add column terms_accepted_at timestamptz,
    add column terms_version     int,
    add column last_export_at    timestamptz;
