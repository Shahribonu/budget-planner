#!/bin/sh
# Nightly pg_dump into /backups, keeping KEEP_DAYS days. Restore with:
#   gunzip -c backups/budget_planner-YYYY-MM-DD.sql.gz | docker compose exec -T db psql -U budget -d budget_planner
set -eu
while true; do
	f="/backups/budget_planner-$(date +%F).sql.gz"
	if pg_dump -h db -U budget -d budget_planner --no-owner | gzip > "$f.tmp"; then
		mv "$f.tmp" "$f"
		echo "backup: wrote $f ($(du -h "$f" | cut -f1))"
	else
		rm -f "$f.tmp"
		echo "backup: FAILED" >&2
	fi
	find /backups -name 'budget_planner-*.sql.gz' -mtime +"$KEEP_DAYS" -delete
	sleep 86400
done
