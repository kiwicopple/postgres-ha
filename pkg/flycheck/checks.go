package flycheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	suite "github.com/fly-examples/postgres-ha/pkg/check"
)

const Port = 5500

func Handler() http.Handler {
	// r := chi.NewRouter()

	r := http.NewServeMux()

	r.HandleFunc("/flycheck/vm", runVMChecks)
	r.HandleFunc("/flycheck/pg", runPGChecks)
	r.HandleFunc("/flycheck/role", runRoleCheck)

	// http.HandleFunc("/flycheck/vm", runVMChecks)
	// http.HandleFunc("/flycheck/pg", runPGChecks)
	// http.HandleFunc("/flycheck/role", runRoleCheck)

	// fmt.Printf("Listening on port %d", Port)
	// http.ListenAndServe(fmt.Sprintf(":%d", Port), nil)

	return r
}

func runVMChecks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (5 * time.Second))
	defer cancel()
	suite := &suite.CheckSuite{Name: "VM"}
	suite = CheckVM(suite)

	go func(ctx context.Context) {
		suite.Process(ctx)
		cancel()
	}(ctx)

	select {
	case <-ctx.Done():
		handleCheckResponse(w, suite, false)
	}
}

func runPGChecks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (5 * time.Second))
	defer cancel()
	suite := &suite.CheckSuite{Name: "PG"}
	suite, err := CheckPostgreSQL(ctx, suite)
	if err != nil {
		suite.ErrOnSetup = err
		cancel()
	}

	go func() {
		suite.Process(ctx)
		cancel()
	}()

	select {
	case <-ctx.Done():
		handleCheckResponse(w, suite, false)
	}
}

func runRoleCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 5))
	defer cancel()

	suite := &suite.CheckSuite{Name: "Role"}
	suite, err := PostgreSQLRole(ctx, suite)
	if err != nil {
		suite.ErrOnSetup = err
		cancel()
	}

	go func() {
		suite.Process(ctx)
		cancel()
	}()

	select {
	case <-ctx.Done():
		handleCheckResponse(w, suite, true)
	}
}

func handleCheckResponse(w http.ResponseWriter, suite *suite.CheckSuite, raw bool) {
	if suite.ErrOnSetup != nil {
		handleError(w, suite.ErrOnSetup)
		return
	}
	var result string
	if raw {
		result = suite.RawResult()
	} else {
		result = suite.Result()
	}
	if !suite.Passed() {
		handleError(w, fmt.Errorf(result))
		return
	}
	json.NewEncoder(w).Encode(result)
}

func handleError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(err.Error())
}
