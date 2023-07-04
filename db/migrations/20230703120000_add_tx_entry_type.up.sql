alter table transaction_entries 
add column entry_type character varying,
add column fee_reserve_id bigint;