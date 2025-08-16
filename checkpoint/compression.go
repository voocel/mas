package checkpoint

import (
	"bytes"
	"compress/gzip"
	"compress/lzw"
	"context"
	"io"
	"math"
	"runtime"
	"sync"
	"time"
)

// CompressionType defines the type of compression to use
type CompressionType string

const (
	CompressionNone CompressionType = "none"
	CompressionGzip CompressionType = "gzip"
	CompressionLZW  CompressionType = "lzw"
	CompressionAuto CompressionType = "auto" // Automatically choose best compression
)

// CompressionConfig contains advanced compression settings
type CompressionConfig struct {
	// Type of compression to use
	Type CompressionType `json:"type"`

	// Compression level (1-9 for gzip, ignored for others)
	Level int `json:"level"`

	// Minimum size threshold for compression (bytes)
	MinSize int64 `json:"min_size"`

	// Maximum compression time allowed
	MaxCompressionTime time.Duration `json:"max_compression_time"`

	// Enable parallel compression for large data
	ParallelCompression bool `json:"parallel_compression"`
}

// DefaultCompressionConfig returns default compression settings
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Type:                CompressionGzip,
		Level:               6,    // Balanced compression
		MinSize:             1024, // 1KB minimum
		MaxCompressionTime:  30 * time.Second,
		ParallelCompression: true,
	}
}

// CompressedData contains compressed data with metadata
type CompressedData struct {
	Data             []byte          `json:"data"`
	CompressionType  CompressionType `json:"compression_type"`
	OriginalSize     int64           `json:"original_size"`
	CompressedSize   int64           `json:"compressed_size"`
	CompressionRatio float64         `json:"compression_ratio"`
	CompressionTime  time.Duration   `json:"compression_time"`
}

// AdvancedCompressor provides advanced compression capabilities
type AdvancedCompressor struct {
	config CompressionConfig
	stats  CompressionStats
	mu     sync.RWMutex
}

// CompressionStats tracks compression performance
type CompressionStats struct {
	TotalOperations     int64         `json:"total_operations"`
	TotalOriginalSize   int64         `json:"total_original_size"`
	TotalCompressedSize int64         `json:"total_compressed_size"`
	AverageRatio        float64       `json:"average_ratio"`
	AverageTime         time.Duration `json:"average_time"`
	LastUpdated         time.Time     `json:"last_updated"`
}

// NewAdvancedCompressor creates a new advanced compressor
func NewAdvancedCompressor(config CompressionConfig) *AdvancedCompressor {
	return &AdvancedCompressor{
		config: config,
		stats: CompressionStats{
			LastUpdated: time.Now(),
		},
	}
}

// Compress compresses data using the configured method
func (ac *AdvancedCompressor) Compress(ctx context.Context, data []byte) (*CompressedData, error) {
	startTime := time.Now()
	originalSize := int64(len(data))

	// Check minimum size threshold
	if originalSize < ac.config.MinSize {
		return &CompressedData{
			Data:             data,
			CompressionType:  CompressionNone,
			OriginalSize:     originalSize,
			CompressedSize:   originalSize,
			CompressionRatio: 1.0,
			CompressionTime:  time.Since(startTime),
		}, nil
	}

	compressCtx, cancel := context.WithTimeout(ctx, ac.config.MaxCompressionTime)
	defer cancel()

	var compressed []byte
	var err error
	compressionType := ac.config.Type

	if ac.config.Type == CompressionAuto {
		compressionType = ac.selectBestCompression(data)
	}

	switch compressionType {
	case CompressionGzip:
		compressed, err = ac.compressGzip(compressCtx, data)
	case CompressionLZW:
		compressed, err = ac.compressLZW(compressCtx, data)
	case CompressionNone:
		compressed = data
	default:
		compressed, err = ac.compressGzip(compressCtx, data)
	}

	if err != nil {
		compressed = data
		compressionType = CompressionNone
	}

	compressedSize := int64(len(compressed))
	compressionTime := time.Since(startTime)
	ratio := float64(compressedSize) / float64(originalSize)

	// Update statistics
	ac.updateStats(originalSize, compressedSize, compressionTime)

	result := &CompressedData{
		Data:             compressed,
		CompressionType:  compressionType,
		OriginalSize:     originalSize,
		CompressedSize:   compressedSize,
		CompressionRatio: ratio,
		CompressionTime:  compressionTime,
	}

	return result, nil
}

// Decompress decompresses data
func (ac *AdvancedCompressor) Decompress(ctx context.Context, compressedData *CompressedData) ([]byte, error) {
	if compressedData.CompressionType == CompressionNone {
		return compressedData.Data, nil
	}

	switch compressedData.CompressionType {
	case CompressionGzip:
		return ac.decompressGzip(ctx, compressedData.Data)
	case CompressionLZW:
		return ac.decompressLZW(ctx, compressedData.Data)
	default:
		return compressedData.Data, nil
	}
}

// compressGzip compresses data using gzip
func (ac *AdvancedCompressor) compressGzip(ctx context.Context, data []byte) ([]byte, error) {
	var buf bytes.Buffer

	if ac.config.ParallelCompression && len(data) > 1024*1024 { // 1MB threshold
		return ac.compressGzipParallel(ctx, data)
	}

	gw, err := gzip.NewWriterLevel(&buf, ac.config.Level)
	if err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	go func() {
		_, err := gw.Write(data)
		if err != nil {
			done <- err
			return
		}
		done <- gw.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case <-ctx.Done():
		gw.Close()
		return nil, ctx.Err()
	}
}

// compressGzipParallel compresses large data in parallel chunks
func (ac *AdvancedCompressor) compressGzipParallel(ctx context.Context, data []byte) ([]byte, error) {
	chunkSize := len(data) / runtime.NumCPU()
	if chunkSize < 64*1024 { // Minimum 64KB per chunk
		chunkSize = 64 * 1024
	}

	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}

	// Compress chunks in parallel
	compressedChunks := make([][]byte, len(chunks))
	errChan := make(chan error, len(chunks))

	for i, chunk := range chunks {
		go func(idx int, data []byte) {
			var buf bytes.Buffer
			gw, err := gzip.NewWriterLevel(&buf, ac.config.Level)
			if err != nil {
				errChan <- err
				return
			}

			if _, err := gw.Write(data); err != nil {
				errChan <- err
				return
			}

			if err := gw.Close(); err != nil {
				errChan <- err
				return
			}

			compressedChunks[idx] = buf.Bytes()
			errChan <- nil
		}(i, chunk)
	}

	// Wait for all chunks to complete
	for i := 0; i < len(chunks); i++ {
		select {
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Combine compressed chunks
	var result bytes.Buffer
	for _, chunk := range compressedChunks {
		result.Write(chunk)
	}

	return result.Bytes(), nil
}

// decompressGzip decompresses gzip data
func (ac *AdvancedCompressor) decompressGzip(ctx context.Context, data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	done := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, gr); err != nil {
			errChan <- err
			return
		}
		done <- buf.Bytes()
	}()

	select {
	case result := <-done:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// compressLZW compresses data using LZW
func (ac *AdvancedCompressor) compressLZW(ctx context.Context, data []byte) ([]byte, error) {
	var buf bytes.Buffer
	lw := lzw.NewWriter(&buf, lzw.MSB, 8)

	done := make(chan error, 1)
	go func() {
		_, err := lw.Write(data)
		if err != nil {
			done <- err
			return
		}
		done <- lw.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case <-ctx.Done():
		lw.Close()
		return nil, ctx.Err()
	}
}

// decompressLZW decompresses LZW data
func (ac *AdvancedCompressor) decompressLZW(ctx context.Context, data []byte) ([]byte, error) {
	lr := lzw.NewReader(bytes.NewReader(data), lzw.MSB, 8)
	defer lr.Close()

	done := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, lr); err != nil {
			errChan <- err
			return
		}
		done <- buf.Bytes()
	}()

	select {
	case result := <-done:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// selectBestCompression automatically selects the best compression method
func (ac *AdvancedCompressor) selectBestCompression(data []byte) CompressionType {
	// Simple heuristic based on data characteristics
	if len(data) < 1024 {
		return CompressionNone
	}

	// Sample first 1KB to determine data characteristics
	sampleSize := 1024
	if len(data) < sampleSize {
		sampleSize = len(data)
	}
	sample := data[:sampleSize]

	// Calculate entropy (simplified)
	entropy := ac.calculateEntropy(sample)

	// Choose compression based on entropy
	if entropy > 7.5 { // High entropy (random data)
		return CompressionNone
	} else if entropy > 6.0 { // Medium entropy
		return CompressionLZW
	} else { // Low entropy (structured data)
		return CompressionGzip
	}
}

// calculateEntropy calculates Shannon entropy of data
func (ac *AdvancedCompressor) calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	freq := make(map[byte]int)
	for _, b := range data {
		freq[b]++
	}

	entropy := 0.0
	length := float64(len(data))

	for _, count := range freq {
		if count > 0 {
			p := float64(count) / length
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// updateStats updates compression statistics
func (ac *AdvancedCompressor) updateStats(originalSize, compressedSize int64, compressionTime time.Duration) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.stats.TotalOperations++
	ac.stats.TotalOriginalSize += originalSize
	ac.stats.TotalCompressedSize += compressedSize

	if ac.stats.TotalOriginalSize > 0 {
		ac.stats.AverageRatio = float64(ac.stats.TotalCompressedSize) / float64(ac.stats.TotalOriginalSize)
	}

	// Running average for compression time
	if ac.stats.TotalOperations == 1 {
		ac.stats.AverageTime = compressionTime
	} else {
		alpha := 0.1
		ac.stats.AverageTime = time.Duration(float64(ac.stats.AverageTime)*(1-alpha) + float64(compressionTime)*alpha)
	}

	ac.stats.LastUpdated = time.Now()
}

// GetStats returns current compression statistics
func (ac *AdvancedCompressor) GetStats() CompressionStats {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.stats
}

// ResetStats resets compression statistics
func (ac *AdvancedCompressor) ResetStats() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.stats = CompressionStats{
		LastUpdated: time.Now(),
	}
}
