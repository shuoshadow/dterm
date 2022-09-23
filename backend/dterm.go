package main

import (
	"encoding/json"
	"os"

	"backend/server"
	"backend/utils"
)

func main() {
	//get k8s config from env
	c := os.Getenv("K8S_CLUSTER_LIST")
	k := utils.K8sConfigList{}
	err := json.Unmarshal([]byte(c), &k)
	if err != nil {
		panic(err.Error())
	}

	//init logger config
	utils.LoggerInit()

	// utils.GetK8sPods()
	for _, config := range k {
		go server.Sync(config)
	}

	server.Run("8081")
}
