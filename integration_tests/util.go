package integration_tests

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/migrations"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/uptrace/bun/migrate"
)

func LndHubTestServiceInit() (svc *service.LndhubService, err error) {
	// change this if you want to run tests using sqlite
	//dbUri := "file:data_test.db"
	//make sure the datbase is empty every time you run the test suite
	dbUri := "postgresql://user:password@localhost/lndhub?sslmode=disable"
	c := &service.Config{
		DatabaseUri:    dbUri,
		JWTSecret:      []byte("SECRET"),
		JWTExpiry:      3600,
		LNDAddress:     "rpc.lnd1.regtest.getalby.com:443",
		LNDMacaroonHex: "0201036c6e6402f801030a10e2133a1cac2c5b4d56e44e32dc64c8551201301a160a0761646472657373120472656164120577726974651a130a04696e666f120472656164120577726974651a170a08696e766f69636573120472656164120577726974651a210a086d616361726f6f6e120867656e6572617465120472656164120577726974651a160a076d657373616765120472656164120577726974651a170a086f6666636861696e120472656164120577726974651a160a076f6e636861696e120472656164120577726974651a140a057065657273120472656164120577726974651a180a067369676e6572120867656e657261746512047265616400000620c4f9783e0873fa50a2091806f5ebb919c5dc432e33800b401463ada6485df0ed",
	}
	dbConn, err := db.Open(c.DatabaseUri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	ctx := context.Background()
	migrator := migrate.NewMigrator(dbConn, migrations.Migrations)
	err = migrator.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init migrations: %w", err)
	}
	_, err = migrator.Migrate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     c.LNDAddress,
		MacaroonHex: c.LNDMacaroonHex,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize lnd service client: %w", err)
	}

	logger := lib.Logger(c.LogFilePath)
	svc = &service.LndhubService{
		Config:    c,
		DB:        dbConn,
		LndClient: lndClient,
		Logger:    logger,
	}
	getInfo, err := lndClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		logger.Fatalf("Error getting node info: %v", err)
	}
	hexPubkey, err := hex.DecodeString(getInfo.IdentityPubkey)
	if err != nil {
		logger.Fatalf("Failed to decode IdentityPubkey: %v", err)
	}
	identityPubKey, err := btcec.ParsePubKey(hexPubkey[:], btcec.S256())
	if err != nil {
		logger.Fatalf("Failed to parse node IdentityPubkey: %v", err)
	}
	svc.IdentityPubkey = identityPubKey

	return svc, nil
}

func createUsers(svc *service.LndhubService, usersToCreate int) (logins []controllers.CreateUserResponseBody, tokens []string, err error) {
	logins = []controllers.CreateUserResponseBody{}
	tokens = []string{}
	for i := 0; i < usersToCreate; i++ {
		user, err := svc.CreateUser(context.Background())
		if err != nil {
			return nil, nil, err
		}
		var login controllers.CreateUserResponseBody
		login.Login = user.Login
		login.Password = user.Password
		logins = append(logins, login)
		token, _, err := svc.GenerateToken(context.Background(), login.Login, login.Password, "")
		if err != nil {
			return nil, nil, err
		}
		tokens = append(tokens, token)
	}
	return logins, tokens, nil
}
