package setup

import (
	"flag"

	"github.com/joho/godotenv"
)

func init() {
	var envFile string
	flag.StringVar(&envFile, "env", ".env", "Path to the .env file")
	flag.Parse()

	loadEnv(envFile)
}

func loadEnv(filePath string) error {
	err := godotenv.Load(filePath)
	return err
}
