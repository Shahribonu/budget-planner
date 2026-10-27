-- Beta metrics from the product plan, section 15. Run with psql against the app database.
-- Analytics events never hold amounts or descriptions; these queries read only events and counts.

-- Activation: users who logged a first transaction within 24 h of /start.
select count(*) filter (where f.at <= s.at + interval '24 hours')::float / nullif(count(*), 0) as activation
from events s
left join events f on f.user_id = s.user_id and f.name = 'first_log'
where s.name = 'bot_start';

-- Time to first log: median seconds from /start to the first saved transaction.
select percentile_cont(0.5) within group (order by (props->>'seconds_since_start')::int) as median_seconds,
       count(*) filter (where (props->>'seconds_since_start')::int <= 60)::float / nullif(count(*), 0) as share_under_60s
from events where name = 'first_log' and props ? 'seconds_since_start';

-- D7 / D30 retention: logged anything on day 7 / day 30 after /start.
select
  avg((exists (select 1 from events e where e.user_id = s.user_id and e.name = 'log_saved'
               and e.at::date = s.at::date + 7))::int) as d7,
  avg((exists (select 1 from events e where e.user_id = s.user_id and e.name = 'log_saved'
               and e.at::date = s.at::date + 30))::int) as d30
from events s where s.name = 'bot_start' and s.at < now() - interval '30 days';

-- North star: weekly active loggers (logged on 3+ distinct days in the last 7 days).
select count(*) from (
  select user_id from events
  where name = 'log_saved' and at > now() - interval '7 days'
  group by user_id having count(distinct at::date) >= 3
) w;

-- Capture quality: how categories were decided, and how often users corrected them.
select props->>'category_origin' as origin, count(*) from events where name = 'log_saved' group by 1 order by 2 desc;
select count(*) filter (where name = 'category_changed')::float
       / nullif(count(*) filter (where name = 'log_saved'), 0) as correction_rate
from events;

-- Questions asked instead of auto-saving, by kind.
select props->>'question' as question, count(*) from events where name = 'draft_created' group by 1;

-- AI cost: Haiku calls and tokens in the last 30 days (multiply by current per-token prices).
select count(*) as calls, sum((props->>'input_tokens')::int) as input_tokens, sum((props->>'output_tokens')::int) as output_tokens,
       avg((props->>'hit')::bool::int) as hit_rate
from events where name = 'ai_category' and at > now() - interval '30 days';
