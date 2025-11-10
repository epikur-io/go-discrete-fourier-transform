
# Discrete Fourier Transform Example (Go + Gonum)

This project demonstrates how to perform a **Discrete Fourier Transform (DFT)** using the [Gonum DSP Fourier](https://pkg.go.dev/gonum.org/v1/gonum/dsp/fourier) package in Go.
It generates a **composite waveform** composed of multiple sine waves, applies a **Hanning window** to reduce spectral leakage, computes the **FFT**, and detects the **main frequency components**.

## Overview

This example showcases:
- Generating a **composite signal** (sum of multiple sine waves).
- Applying a **Hanning window** to smooth discontinuities at signal edges.
- Computing the **Fast Fourier Transform (FFT)** using Gonum’s `fourier` package.
- Identifying the **dominant frequencies** and their magnitudes.

The program reconstructs the frequencies and amplitudes that make up a mixed signal.

<p align="center">
    <img width="600" src="./example_reconstructed_waves.jpeg"><br>
    Source: <a href="https://www.3blue1brown.com/lessons/fourier-transforms#adding-sounds-waves" title="3blue1brown - fourier transforms lesson">https://www.3blue1brown.com/lessons/fourier-transforms</a>
</p>

---

## Requirements

- Go 1.18+
- Gonum

## How It Works

### 1. Generate Composite Wave

```go
wave := GenerateCompositeWave(freqs, amplitudes, sampleRate, duration)
```

### 2. Apply Hanning Window

```go
ApplyHanningWindow(wave)
```

### 3. Compute FFT

```go
fft := fourier.NewFFT(fftSize)
spectrum := fft.Coefficients(nil, paddedWave)
```

### 4. Compute Magnitude Spectrum

Magnitude is calculated from the complex coefficients and scaled by signal length and window gain.

```go
// Compute magnitude spectrum using original wave length for amplitude scaling
windowGain := 0.5
mag := make([]float64, fftSize/2)
for i := 0; i < fftSize/2; i++ {
    mag[i] = cmplx.Abs(spectrum[i]) * 2 / float64(len(wave)) / windowGain
}
```

### 5. Find Main Peaks

```go
peaks := FindMainPeaks(mag, freqRes, neighborhoodHz, threshold)
```

### Parameters

| Parameter        | Description                             | Example Value     |
|------------------|------------------------------------------|-------------------|
| `sampleRate`     | Sampling rate in Hz                      | `1024`            |
| `duration`       | Signal duration in seconds               | `15.0`            |
| `freqs`          | Frequencies of the sine components (Hz)  | `[50, 120, 300]`  |
| `amplitudes`     | Amplitudes of each sine wave             | `[1.0, 0.5, 0.8]` |
| `neighborhoodHz` | Range for filtering side lobes (Hz)      | `3.0`             |
| `threshold`      | Minimum magnitude for peak detection     | `0.05`            |

### Example program output

```
$ go run examples/synthetic/dft_synthetic.go

Detected main frequencies:
Frequency: 50.0 Hz, Magnitude: 1.000
Frequency: 120.0 Hz, Magnitude: 0.500
Frequency: 300.0 Hz, Magnitude: 0.800
```

Or for an audio file:

```
$ go run examples/audio_file/dft_audio_file.go \
    -input my_audio_file.mp3 \
    -duration 1 \
    -mmt 0.001 \
    -start 0
```