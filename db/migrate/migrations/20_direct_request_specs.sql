-- +goose Up
ALTER TABLE direct_request_specs ADD COLUMN num_confirmations bigint DEFAULT NULL;

ALTER TABLE direct_request_specs ADD COLUMN requesters TEXT;

ALTER TABLE direct_request_specs ADD COLUMN min_contract_payment numeric(78,0);

ALTER TABLE direct_request_specs RENAME COLUMN num_confirmations TO min_incoming_confirmations;

-- +goose Down
ALTER TABLE direct_request_specs DROP COLUMN num_confirmations;

ALTER TABLE direct_request_specs DROP COLUMN requesters;

ALTER TABLE direct_request_specs DROP COLUMN min_contract_payment;

ALTER TABLE direct_request_specs RENAME COLUMN min_incoming_confirmations TO num_confirmations;