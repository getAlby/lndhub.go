package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/service"
)

type Daemon struct {
	svc *service.LndhubService
}

func Run(svc *service.LndhubService) {
	daemon := &Daemon{
		svc: svc,
	}
	http.HandleFunc("/users", daemon.userHandler)
	fmt.Println("Starting plugin: admin server")
	http.ListenAndServe(":8081", nil)
}

func (daemon *Daemon) userHandler(w http.ResponseWriter, r *http.Request) {
	users, err := daemon.SelectAllUsers(context.Background())
	if err != nil {
		w.Write([]byte(fmt.Sprintf("something went wrong: %s", err.Error())))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (daemon *Daemon) SelectAllUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User

	query := daemon.svc.DB.NewSelect().Model(&users)

	err := query.Scan(ctx)
	if err != nil {
		return nil, err
	}
	return users, nil
}
