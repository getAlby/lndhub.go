package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {

		if db.Dialect().Name().String() != "pg" {
			fmt.Printf("\033[1;31m%s\033[0m", "You are not using PostgreSQL. DB level checks can not be enabled!\n")
			return nil
		}
		sql := `
			-- make sure transfers happen from one account to another one
				alter table transaction_entries
				ADD CONSTRAINT check_not_same_account
				CHECK (debit_account_id != credit_account_id);

			-- make sure that account balances >= 0 (except for incoming account)
				CREATE OR REPLACE FUNCTION check_balance()
					RETURNS TRIGGER AS $$
				DECLARE
					sum BIGINT;
					debit_account_type VARCHAR;
				BEGIN
					SELECT INTO debit_account_type type
					FROM accounts
					WHERE id = NEW.debit_account_id;

					SELECT INTO  sum SUM(amount)
					FROM account_ledgers
					WHERE account_ledgers.account_id = NEW.debit_account_id;

					-- the incoming account can have a negative balance
					-- all other accounts must have a positive balance
					IF sum < 0 AND debit_account_type != 'incoming'
					THEN
						RAISE EXCEPTION 'invalid balance [user_id:%] [debit_account_id:%] balance [%]',
						NEW.user_id,
						NEW.debit_account_id,
						sum;
					END IF;
					RETURN NEW;
				END;
				$$ LANGUAGE plpgsql;
				CREATE TRIGGER check_balance
				AFTER INSERT OR UPDATE ON transaction_entries
				FOR EACH ROW EXECUTE PROCEDURE check_balance();
		`
		if _, err := db.Exec(sql); err != nil {
			return err
		}
		return nil
	}, nil)
}
