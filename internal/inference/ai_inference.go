package inference

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

type WhisperInferenceResult struct {
	CPU            float64
	RAM            uint64
	InferenceTime  float64
	RealTimeFactor float64
	AudioDuration  float64
}

func CollectWhisperMetrics(
	pid int,
	elapsed time.Duration,
	audioDuration float64,
) (WhisperInferenceResult, error) {

	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return WhisperInferenceResult{}, err
	}

	cpu, err := p.CPUPercent()
	if err != nil {
		return WhisperInferenceResult{}, err
	}

	memory, err := p.MemoryInfo()
	if err != nil {
		return WhisperInferenceResult{}, err
	}

	transcriptionTime := elapsed.Seconds()

	if audioDuration <= 0 {
		return WhisperInferenceResult{}, fmt.Errorf("audio duration must be greater than zero")
	}

	rtf := transcriptionTime / audioDuration

	return WhisperInferenceResult{
		CPU:            cpu,
		RAM:            memory.RSS,
		InferenceTime:  transcriptionTime,
		RealTimeFactor: rtf,
		AudioDuration:  audioDuration,
	}, nil
}
