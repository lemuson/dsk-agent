#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)

apply_sql() {
  docker compose --project-directory "$PROJECT_DIR" exec -T database \
    sh -c 'psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$DB_NAME"' < "$1"
}

apply_sql "$PROJECT_DIR/backend/migrations/000007_reconcile_schema_and_notification_state.sql"
apply_sql "$PROJECT_DIR/backend/migrations/000008_sales_workflow.sql"
apply_sql "$PROJECT_DIR/backend/migrations/000009_offer_ancillary_price_snapshot.sql"
apply_sql "$PROJECT_DIR/backend/migrations/000010_chat_delete_and_lifecycle_updates.sql"
apply_sql "$PROJECT_DIR/backend/seeds/demo.sql"

echo "Demo seed applied. Staff accounts: manager@dsk.demo, supervisor@dsk.demo"
