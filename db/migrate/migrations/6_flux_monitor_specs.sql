-- +goose Up
ALTER TABLE flux_monitor_specs
ADD min_payment varchar(255);

ALTER TABLE flux_monitor_specs DROP COLUMN precision;

-- +goose Down
ALTER TABLE flux_monitor_specs
DROP COLUMN min_payment;

ALTER TABLE flux_monitor_specs ADD COLUMN precision integer;
