package main

import (
	"stock_adapter/server"
)

func main() {

	s := server.NewServer()
	s.Start(":8000")
}
