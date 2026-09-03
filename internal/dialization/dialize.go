package dialization

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

func RunDialization(audioPath string) ([]Segment, error) {
	hf_token := os.Getenv("HF_TOKEN")
	cmd := exec.Command("py", "python/dialize.py", audioPath, hf_token)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("dialization failed: %v — stderr: %s", err, stderr.String())
	}

	var dializedSegments []Segment
	if err := json.Unmarshal(out, &dializedSegments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dialized segments: %v", err)
	}
	return dializedSegments, nil
}
