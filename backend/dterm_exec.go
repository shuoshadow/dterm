package main

import (
	"backend/server"
	"backend/utils"
)

func main() {
	//init logger config
	utils.LoggerInit()

	server.ExecServerRun("2222")
}
