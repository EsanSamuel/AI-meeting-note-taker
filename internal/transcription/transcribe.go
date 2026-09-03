package transcription

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/AssemblyAI/assemblyai-go-sdk"
)

func ExtractTextFromAudio(id string, reader io.Reader) (string, error) {
	ASSEMBLYAI_API := os.Getenv("ASSEMBLYAI_API_KEY")
	if ASSEMBLYAI_API == "" {
		return "", fmt.Errorf("ASSEMBLYAI_API_KEY is not set in environment variables")
	}
	client := assemblyai.NewClient(ASSEMBLYAI_API)

	transcript, err := client.Transcripts.TranscribeFromReader(
		context.Background(),
		reader,
		nil,
	)
	if err != nil {
		return "", err
	}

	fmt.Println(*transcript.Text)
	return *transcript.Text, nil
}

/*text, err := transcription.ExtractTextFromAudio(recording.ID, bytes.NewReader(audio.Data))
if err != nil {
	c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	return
}*/
