package api

import "github.com/gorilla/mux"

// registerSwagger is a no-op unless built with -tags swagger
func registerSwagger(r *mux.Router) {}
