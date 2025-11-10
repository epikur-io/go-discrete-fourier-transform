package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/cmplx"

	"os"
	"time"

	"gonum.org/v1/gonum/dsp/fourier"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
)

// LoadAudioAsFloat64 returns mono samples in [-1..1], inferred sample rate (Hz), and audio duration.
func LoadAudioAsFloat64(path string) (mono []float64, sampleRate int, duration time.Duration, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()

	var streamer beep.StreamSeekCloser
	var format beep.Format

	switch {
	case hasExt(path, ".wav"):
		streamer, format, err = wav.Decode(f)
	case hasExt(path, ".mp3"):
		streamer, format, err = mp3.Decode(f)
	case hasExt(path, ".ogg"):
		streamer, format, err = vorbis.Decode(f)
	default:
		return nil, 0, 0, fmt.Errorf("unsupported format")
	}
	if err != nil {
		return nil, 0, 0, err
	}
	defer streamer.Close()

	// buffer of stereo frames
	buf := make([][2]float64, 4096)

	for {
		n, ok := streamer.Stream(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				// mix stereo -> mono (average). If source is mono, second channel is 0.
				m := (buf[i][0] + buf[i][1]) / 2
				mono = append(mono, m)
			}
		}
		if !ok {
			break
		}
	}

	// two ways to get sample rate as integer:
	sampleRateFromN := format.SampleRate.N(time.Second) // uses N(d time.Duration)
	sampleRateFromCast := int(format.SampleRate)        // direct cast

	if sampleRateFromN != sampleRateFromCast {
		// they should be equal; choose cast as the canonical integer
	}
	sampleRate = sampleRateFromCast

	// compute duration from number of frames (mono length) and sample rate
	duration = time.Duration(len(mono)) * time.Second / time.Duration(sampleRate)

	return mono, sampleRate, duration, nil
}

func hasExt(path, ext string) bool {
	if len(path) < len(ext) {
		return false
	}
	return path[len(path)-len(ext):] == ext
}

// Example of a discrete fourier transform.
// This example shows the reconstruction of the individual frequencies and their magnitudes based of a composite wave.

// GenerateCompositeWave generates a sum of sine waves
func GenerateCompositeWave(freqs, amplitudes []float64, sampleRate int, duration float64) []float64 {
	nSamples := int(float64(sampleRate) * duration)
	wave := make([]float64, nSamples)

	for i := 0; i < nSamples; i++ {
		t := float64(i) / float64(sampleRate)
		for j, freq := range freqs {
			wave[i] += amplitudes[j] * math.Sin(2*math.Pi*freq*t)
		}
	}
	return wave
}

// ApplyHanningWindow applies a Hanning window to reduce spectral leakage
func ApplyHanningWindow(wave []float64) {
	N := len(wave)
	for i := 0; i < N; i++ {
		wave[i] *= 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(N-1)))
	}
}

// FindMainPeaks detects main frequency peaks and filters side lobes
func FindMainPeaks(mag []float64, freqRes float64, neighborhoodHz float64, threshold float64) []int {
	peaks := []int{}
	binRadius := int(neighborhoodHz / freqRes)

	for i := 1; i < len(mag)-1; i++ {
		if mag[i] < threshold {
			continue
		}

		isMax := true
		start := i - binRadius
		if start < 0 {
			start = 0
		}
		end := i + binRadius
		if end >= len(mag) {
			end = len(mag) - 1
		}

		for j := start; j <= end; j++ {
			if mag[j] > mag[i] {
				isMax = false
				break
			}
		}

		if isMax {
			peaks = append(peaks, i)
			i = end // skip neighborhood
		}
	}

	return peaks
}

func main() {
	inputFile := flag.String("input", "", "path for input audio file")
	inputDurationSecs := flag.Float64("duration", 1, "duration in seconds")
	startAt := flag.Float64("start", 0, "location to start in the audio signal (in seconds)")
	minMagThreshold := flag.Float64("mmt", 0.5, "Min. magnitude threshold (for detecting main peaks)")
	flag.Parse()

	fmt.Println(math.Max(1, 2))

	if *inputFile == "" {
		log.Fatalln("missing input file")
	}

	// Generate wave
	// wave := GenerateCompositeWave(freqs, amplitudes, sampleRate, duration)
	wave, sampleRate, audioDur, err := LoadAudioAsFloat64(*inputFile)
	if err != nil {
		log.Fatalln("failed to load audio file:", err)
	}
	log.Println("input audio duration:", audioDur)
	log.Println("sampleRate:", sampleRate)
	log.Println("audioDur/sampleRate:", *inputDurationSecs*float64(sampleRate))
	log.Println("wave length:", len(wave))
	log.Println("wave start:", int((*startAt)*float64(sampleRate)))
	log.Println("wave end:", int(*inputDurationSecs*float64(sampleRate)))

	// sanity check
	if len(wave) < int((*startAt)*float64(sampleRate)) {
		log.Fatalf("invalid starting point in wave. lenght is %d but starting point is %d", len(wave), int((*startAt)*float64(sampleRate)))
	}
	if len(wave) < int((*startAt)*float64(sampleRate))+int(*inputDurationSecs*float64(sampleRate)) {
		log.Fatalf("invalid end point in wave. lenght is %d but end point is %d", len(wave), int((*startAt)*float64(sampleRate)))
	}

	wave = wave[int((*startAt)*float64(sampleRate)) : int((*startAt)*float64(sampleRate))+int(*inputDurationSecs*float64(sampleRate))]
	// Apply Hanning window
	ApplyHanningWindow(wave)

	// Determine FFT size as next power of 2
	nSamples := len(wave)
	fftSize := 1
	for fftSize < nSamples {
		fftSize *= 2
	}

	// Zero-pad
	paddedWave := make([]float64, fftSize)
	copy(paddedWave, wave)

	// Compute FFT
	fft := fourier.NewFFT(fftSize)
	spectrum := fft.Coefficients(nil, paddedWave)

	// Compute magnitude spectrum using original wave length for amplitude scaling
	windowGain := 0.5
	mag := make([]float64, fftSize/2)
	for i := 0; i < fftSize/2; i++ {
		mag[i] = cmplx.Abs(spectrum[i]) * 2 / float64(len(wave)) / windowGain
	}

	// Frequency resolution
	freqRes := float64(sampleRate) / float64(fftSize)
	neighborhoodHz := 3.0 // filter side lobes ±3Hz

	// Find main peaks
	peaks := FindMainPeaks(mag, freqRes, neighborhoodHz, *minMagThreshold)

	// Print results
	fmt.Println("Detected main frequencies:")
	for _, i := range peaks {
		freq := float64(i) * float64(sampleRate) / float64(fftSize)
		fmt.Printf("Frequency: %.2f Hz, Magnitude: %.8f\n", freq, mag[i])
	}
}
