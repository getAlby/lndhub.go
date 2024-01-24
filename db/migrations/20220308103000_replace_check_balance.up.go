package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)
// TODO ensure that AssetID has been accounted for here
func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {

		if db.Dialect().Name().String() != "pg" {
			fmt.Printf("\033[1;31m%s\033[0m", "You are not using PostgreSQL. DB level checks can not be enabled!\n")
			return nil
		}
		sql := `
			-- make sure that account balances >= 0 (except for incoming and fees accounts)
				CREATE OR REPLACE FUNCTION check_balance()
					RETURNS TRIGGER AS $$
				DECLARE
					sum BIGINT;
					debit_account_type VARCHAR;
					credit_account_type VARCHAR;
				BEGIN

					-- LOCK the account if the transaction is not from an incoming account
					--  This makes sure we always check the balance of the account before commiting a transaction
					--  (incoming accounts can be negative, so we do not care about those)
					SELECT INTO debit_account_type type
					FROM accounts
					WHERE id = NEW.debit_account_id AND type <> 'incoming'
					-- IMPORTANT: lock rows but do not wait for another lock to be released.
					--   Waiting would result in a deadlock because two parallel transactions could try to lock the same rows
					--   NOWAIT reports an error rather than waiting for the lock to be released
					--   This can happen when two transactions try to access the same account
					FOR UPDATE NOWAIT;

					-- check if credit_account type is fees, if it's fees we don't check for negative balance constraint
					SELECT INTO credit_account_type type
					FROM accounts
					WHERE id = NEW.credit_account_id AND type <> 'fees'
					-- IMPORTANT: lock rows but do not wait for another lock to be released.
					--   Waiting would result in a deadlock because two parallel transactions could try to lock the same rows
					--   NOWAIT reports an error rather than waiting for the lock to be released
					--   This can happen when two transactions try to access the same account
					FOR UPDATE NOWAIT;

					-- If it is an debit incoming account or fees credit account return; otherwise check the balance
					IF debit_account_type IS NULL OR credit_account_type IS NULL
					THEN
						RETURN NEW;
					END IF;

					-- Calculate the account balance
					SELECT INTO sum SUM(amount)
					FROM account_ledgers
					WHERE account_ledgers.account_id = NEW.debit_account_id;

					-- IF the account would go negative raise an exception
					IF sum < 0
					THEN
						RAISE EXCEPTION 'invalid balance [user_id:%] [debit_account_id:%] balance [%]',
						NEW.user_id,
						NEW.debit_account_id,
						sum;
					END IF;
					RETURN NEW;
				END;
				$$ LANGUAGE plpgsql;

				-- first we drop trigger and re-add it again with modified function
				DROP TRIGGER IF EXISTS check_balance ON transaction_entries;

				-- create deferrable trigger which is executed at the end of the transaction to check the balance for each inserted transaction entry
				CREATE CONSTRAINT TRIGGER check_balance
				AFTER INSERT OR UPDATE ON transaction_entries
				DEFERRABLE
				FOR EACH ROW EXECUTE PROCEDURE check_balance();
		`
		if _, err := db.Exec(sql); err != nil {
			return err
		}
		return nil
	}, nil)
}
