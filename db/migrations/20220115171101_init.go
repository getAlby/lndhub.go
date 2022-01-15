package migrations

import (
	"context"

	"github.com/bumi/lndhub.go/db/models"
	"github.com/uptrace/bun"
)

/* Since this init will reflect the latest model fields when run on fresh db
make sure that when you add/remove columns in subsequent migrations IfNotExists/IfExists is used
otherwise it's going to result in errors.

Once this has been deployed some place - we need to start introducing subsequent migrations.
*/
func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {

		if _, err := db.NewCreateTable().Model((*models.User)(nil)).Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*models.Invoice)(nil)).Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*models.Account)(nil)).Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*models.TransactionEntry)(nil)).Exec(ctx); err != nil {
			return err
		}

		return nil
	}, nil)
}
