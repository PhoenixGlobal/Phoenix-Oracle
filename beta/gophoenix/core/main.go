package main

import (
	"PhoenixOracle/gophoenix/core/logger"
	"PhoenixOracle/gophoenix/core/services"
	"PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/web"
	"log"
)

func main() {
	app := services.NewApplication(store.NewConfig())

	services.Authenticate(app.Store)
	r := web.Router(app)
	err := app.Start()
	if err != nil{
		log.Fatal(err)
	}
	defer app.Stop()

	logger.Fatal(r.Run())
}