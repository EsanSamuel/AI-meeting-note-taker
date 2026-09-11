package config

import (
	"context"
	"log"
	"os"

	"google.golang.org/genai"
)

func Ai(prompt string) (string, error) {
	ctx := context.Background()

	api_key := os.Getenv("GEMINI_API_KEY")
	config := &genai.ClientConfig{
		APIKey: api_key,
	}
	client, err := genai.NewClient(ctx, config)
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.Models.GenerateContent(
		ctx,
		"gemini-3.5-flash",
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		log.Println(err)
		return "", err
	}
	//fmt.Println(result.Text())
	return result.Text(), nil
}
