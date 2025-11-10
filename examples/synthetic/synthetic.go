package main

import (
	"fmt"
	"math"
	"math/cmplx"

	"gonum.org/v1/gonum/dsp/fourier"
)

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
	// Parameters
	sampleRate := 1024
	duration := 15.0 // arbitrary duration > 10s
	freqs := []float64{50, 120, 300}
	amplitudes := []float64{1.0, 0.5, 0.8}

	// Generate wave
	wave := GenerateCompositeWave(freqs, amplitudes, sampleRate, duration)

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
	threshold := 0.05     // minimum magnitude

	// Find main peaks
	peaks := FindMainPeaks(mag, freqRes, neighborhoodHz, threshold)

	// Print results
	fmt.Println("Detected main frequencies:")
	for _, i := range peaks {
		freq := float64(i) * float64(sampleRate) / float64(fftSize)
		fmt.Printf("Frequency: %.1f Hz, Magnitude: %.3f\n", freq, mag[i])
	}
}
