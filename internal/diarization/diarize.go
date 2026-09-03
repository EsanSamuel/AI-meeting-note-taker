package diarization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type Segment struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Speaker string  `json:"speaker"`
}

func RunDiarization(audioPath string) ([]Segment, error) {
	hf_token := os.Getenv("HF_TOKEN")
	cmd := exec.Command("py", "python/diarize.py", audioPath, hf_token)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("diarization failed: %v — stderr: %s", err, stderr.String())
	}

	var diarizedSegments []Segment
	if err := json.Unmarshal(out, &diarizedSegments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal diarized segments: %v", err)
	}
	return diarizedSegments, nil
}
