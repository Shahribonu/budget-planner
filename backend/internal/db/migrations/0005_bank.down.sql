alter table drafts drop constraint drafts_source_check;
alter table drafts add constraint drafts_source_check check (source in ('chat', 'voice'));
alter table drafts drop constraint drafts_question_check;
alter table drafts add constraint drafts_question_check check (question in ('account', 'amount', 'duplicate'));
drop index accounts_card_uniq;
alter table accounts drop column card_mask, drop column bank_balance, drop column bank_balance_at;
