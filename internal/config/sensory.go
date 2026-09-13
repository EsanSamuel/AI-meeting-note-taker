package config

import (
	"fmt"
	"os"

	logClient "github.com/EsanSamuel/sensory/LogClient"
)

func InitLogger() *logClient.Client {
	api_key := os.Getenv("SENSORY_API_KEY")

	logger, err := logClient.New(api_key)
	if err != nil {
		fmt.Println("Logger connection failed!", err)
		return logClient.NewNoOp()
	}

	fmt.Println("Logger connected successfully")
	return logger
}