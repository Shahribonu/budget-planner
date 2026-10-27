-- Bank notifications: an account remembers its card (last four digits) so forwarded
-- messages land in the right place. Full card numbers are never stored.
alter table accounts add column card_mask text check (card_mask ~ '^[0-9]{4}$');
-- The newest balance a bank reported for this card, to offer a one-tap reconcile.
alter table accounts add column bank_balance bigint, add column bank_balance_at timestamptz;
create unique index accounts_card_uniq on accounts (user_id, card_mask) where card_mask is not null and archived_at is null;

alter table drafts drop constraint drafts_question_check;
alter table drafts add constraint drafts_question_check check (question in ('account', 'amount', 'duplicate', 'card', 'bank_kind', 'transfer_from'));
alter table drafts drop constraint drafts_source_check;
alter table drafts add constraint drafts_source_check check (source in ('chat', 'voice', 'bank'));
