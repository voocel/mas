package runtime

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Thread represents an execution session
type Thread struct {
	ID        string                 `json:"id"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// StateSnapshot captures a snapshot of state
type StateSnapshot struct {
	ID        string                 `json:"id"`
	ThreadID  string                 `json:"thread_id"`
	Values    map[string]interface{} `json:"values"`
	Next      []string               `json:"next"`
	Config    map[string]interface{} `json:"config"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	ParentID  string                 `json:"parent_id,omitempty"`
}

// CheckpointTuple bundles checkpoint data
type CheckpointTuple struct {
	Config   map[string]interface{} `json:"config"`
	Snapshot *StateSnapshot         `json:"snapshot"`
	Metadata map[string]interface{} `json:"metadata"`
	ParentID string                 `json:"parent_id,omitempty"`
}

// Checkpointer defines the checkpoint interface
type Checkpointer interface {
	// Put stores a checkpoint
	Put(config map[string]interface{}, snapshot *StateSnapshot, metadata map[string]interface{}) error

	// GetTuple retrieves a checkpoint tuple
	GetTuple(config map[string]interface{}) (*CheckpointTuple, error)

	// List enumerates checkpoints
	List(config map[string]interface{}, limit int, before string) ([]*CheckpointTuple, error)

	// PutWrites persists pending writes
	PutWrites(config map[string]interface{}, writes []map[string]interface{}, taskID string) error
}

// ThreadManager defines thread management operations
type ThreadManager interface {
	CreateThread(metadata map[string]interface{}) (*Thread, error)
	GetThread(threadID string) (*Thread, error)
	UpdateThread(threadID string, metadata map[string]interface{}) error
	DeleteThread(threadID string) error
	ListThreads() ([]*Thread, error)
}

// StateSnapshotManager manages state snapshots
type StateSnapshotManager interface {
	SaveSnapshot(snapshot *StateSnapshot) error
	GetSnapshot(snapshotID string) (*StateSnapshot, error)
	GetLatestSnapshot(threadID string) (*StateSnapshot, error)
	GetStateHistory(threadID string) ([]*StateSnapshot, error)
	ForkFromSnapshot(snapshotID string, newThreadID string) (*StateSnapshot, error)
	DeleteSnapshot(snapshotID string) error
}

// MemoryCheckpointer provides an in-memory implementation
type MemoryCheckpointer struct {
	checkpoints map[string]*CheckpointTuple
	writes      map[string][]map[string]interface{}
	mutex       sync.RWMutex
}

// NewMemoryCheckpointer create an in-memory checkpointer
func NewMemoryCheckpointer() *MemoryCheckpointer {
	return &MemoryCheckpointer{
		checkpoints: make(map[string]*CheckpointTuple),
		writes:      make(map[string][]map[string]interface{}),
	}
}

func (m *MemoryCheckpointer) Put(config map[string]interface{}, snapshot *StateSnapshot, metadata map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	threadID, ok := config["thread_id"].(string)
	if !ok {
		return fmt.Errorf("thread_id not found in config")
	}

	if snapshot.ID == "" {
		snapshot.ID = uuid.New().String()
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now()
	}

	tuple := &CheckpointTuple{
		Config:   config,
		Snapshot: snapshot,
		Metadata: metadata,
	}

	m.checkpoints[threadID] = tuple
	return nil
}

func (m *MemoryCheckpointer) GetTuple(config map[string]interface{}) (*CheckpointTuple, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	threadID, ok := config["thread_id"].(string)
	if !ok {
		return nil, fmt.Errorf("thread_id not found in config")
	}

	tuple, exists := m.checkpoints[threadID]
	if !exists {
		return nil, fmt.Errorf("checkpoint not found for thread %s", threadID)
	}

	return tuple, nil
}

func (m *MemoryCheckpointer) List(config map[string]interface{}, limit int, before string) ([]*CheckpointTuple, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var tuples []*CheckpointTuple
	for _, tuple := range m.checkpoints {
		tuples = append(tuples, tuple)
	}

	// Sort by time
	for i := 0; i < len(tuples)-1; i++ {
		for j := i + 1; j < len(tuples); j++ {
			if tuples[i].Snapshot.CreatedAt.Before(tuples[j].Snapshot.CreatedAt) {
				tuples[i], tuples[j] = tuples[j], tuples[i]
			}
		}
	}

	// Apply the limit
	if limit > 0 && len(tuples) > limit {
		tuples = tuples[:limit]
	}

	return tuples, nil
}

func (m *MemoryCheckpointer) PutWrites(config map[string]interface{}, writes []map[string]interface{}, taskID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	threadID, ok := config["thread_id"].(string)
	if !ok {
		return fmt.Errorf("thread_id not found in config")
	}

	key := fmt.Sprintf("%s:%s", threadID, taskID)
	m.writes[key] = writes
	return nil
}

func copyMap(original map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

// Metrics tracks execution metrics
type Metrics struct {
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	Duration      time.Duration          `json:"duration"`
	StepCount     int                    `json:"step_count"`
	ErrorCount    int                    `json:"error_count"`
	SuccessRate   float64                `json:"success_rate"`
	AgentUsage    map[string]int         `json:"agent_usage"`
	ToolUsage     map[string]int         `json:"tool_usage"`
	TokenUsage    TokenUsage             `json:"token_usage"`
	CustomMetrics map[string]interface{} `json:"custom_metrics"`
}

// TokenUsage captures token statistics
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// MetricsCollector accumulates metrics
type MetricsCollector struct {
	metrics Metrics
	mutex   sync.RWMutex
}

// NewMetricsCollector constructs a metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: Metrics{
			StartTime:     time.Now(),
			AgentUsage:    make(map[string]int),
			ToolUsage:     make(map[string]int),
			CustomMetrics: make(map[string]interface{}),
		},
	}
}

func (m *MetricsCollector) RecordAgentUsage(agentID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.metrics.AgentUsage[agentID]++
}

func (m *MetricsCollector) RecordToolUsage(toolName string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.metrics.ToolUsage[toolName]++
}

func (m *MetricsCollector) RecordStep() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.metrics.StepCount++
}

func (m *MetricsCollector) RecordError() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.metrics.ErrorCount++
}

func (m *MetricsCollector) RecordTokenUsage(promptTokens, completionTokens int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.metrics.TokenUsage.PromptTokens += promptTokens
	m.metrics.TokenUsage.CompletionTokens += completionTokens
	m.metrics.TokenUsage.TotalTokens += promptTokens + completionTokens
}

func (m *MetricsCollector) SetCustomMetric(key string, value interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.metrics.CustomMetrics[key] = value
}

func (m *MetricsCollector) GetMetrics() Metrics {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Compute success rate
	if m.metrics.StepCount > 0 {
		m.metrics.SuccessRate = float64(m.metrics.StepCount-m.metrics.ErrorCount) / float64(m.metrics.StepCount)
	}

	m.metrics.EndTime = time.Now()
	m.metrics.Duration = m.metrics.EndTime.Sub(m.metrics.StartTime)

	return Metrics{
		StartTime:     m.metrics.StartTime,
		EndTime:       m.metrics.EndTime,
		Duration:      m.metrics.Duration,
		StepCount:     m.metrics.StepCount,
		ErrorCount:    m.metrics.ErrorCount,
		SuccessRate:   m.metrics.SuccessRate,
		AgentUsage:    copyIntMap(m.metrics.AgentUsage),
		ToolUsage:     copyIntMap(m.metrics.ToolUsage),
		TokenUsage:    m.metrics.TokenUsage,
		CustomMetrics: copyMap(m.metrics.CustomMetrics),
	}
}

func (m *MetricsCollector) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.metrics = Metrics{
		StartTime:     time.Now(),
		AgentUsage:    make(map[string]int),
		ToolUsage:     make(map[string]int),
		CustomMetrics: make(map[string]interface{}),
	}
}

func copyIntMap(original map[string]int) map[string]int {
	copy := make(map[string]int)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

func (m Metrics) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

func (m *Metrics) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}

// CompressedCheckpointer stores checkpoints in compressed form
type CompressedCheckpointer struct {
	checkpoints map[string]*CompressedCheckpointTuple
	writes      map[string][]map[string]interface{}
	mutex       sync.RWMutex
}

// CompressedCheckpointTuple stores compressed checkpoint data
type CompressedCheckpointTuple struct {
	Config         map[string]interface{} `json:"config"`
	CompressedData []byte                 `json:"compressed_data"`
	Metadata       map[string]interface{} `json:"metadata"`
	ParentID       string                 `json:"parent_id,omitempty"`
	Size           int                    `json:"size"`            // Original size
	CompressedSize int                    `json:"compressed_size"` // Compressed size
	CreatedAt      time.Time              `json:"created_at"`
}

// NewCompressedCheckpointer constructs a compressed checkpointer
func NewCompressedCheckpointer() *CompressedCheckpointer {
	return &CompressedCheckpointer{
		checkpoints: make(map[string]*CompressedCheckpointTuple),
		writes:      make(map[string][]map[string]interface{}),
	}
}

func (c *CompressedCheckpointer) Put(config map[string]interface{}, snapshot *StateSnapshot, metadata map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	threadID, ok := config["thread_id"].(string)
	if !ok {
		return fmt.Errorf("thread_id not found in config")
	}

	if snapshot.ID == "" {
		snapshot.ID = uuid.New().String()
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now()
	}

	jsonData, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// Compress data
	compressedData, err := compressData(jsonData)
	if err != nil {
		return fmt.Errorf("failed to compress snapshot: %w", err)
	}

	tuple := &CompressedCheckpointTuple{
		Config:         config,
		CompressedData: compressedData,
		Metadata:       metadata,
		Size:           len(jsonData),
		CompressedSize: len(compressedData),
		CreatedAt:      snapshot.CreatedAt,
	}

	c.checkpoints[threadID] = tuple
	return nil
}

func (c *CompressedCheckpointer) GetTuple(config map[string]interface{}) (*CheckpointTuple, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	threadID, ok := config["thread_id"].(string)
	if !ok {
		return nil, fmt.Errorf("thread_id not found in config")
	}

	compressedTuple, exists := c.checkpoints[threadID]
	if !exists {
		return nil, fmt.Errorf("checkpoint not found for thread %s", threadID)
	}

	// Decompress data
	jsonData, err := decompressData(compressedTuple.CompressedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress snapshot: %w", err)
	}

	var snapshot StateSnapshot
	if err := json.Unmarshal(jsonData, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return &CheckpointTuple{
		Config:   compressedTuple.Config,
		Snapshot: &snapshot,
		Metadata: compressedTuple.Metadata,
		ParentID: compressedTuple.ParentID,
	}, nil
}

func (c *CompressedCheckpointer) List(config map[string]interface{}, limit int, before string) ([]*CheckpointTuple, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var tuples []*CheckpointTuple
	for _, compressedTuple := range c.checkpoints {
		// Decompress data
		jsonData, err := decompressData(compressedTuple.CompressedData)
		if err != nil {
			continue // Skip corrupted data
		}

		var snapshot StateSnapshot
		if err := json.Unmarshal(jsonData, &snapshot); err != nil {
			continue // Skip corrupted data
		}

		tuple := &CheckpointTuple{
			Config:   compressedTuple.Config,
			Snapshot: &snapshot,
			Metadata: compressedTuple.Metadata,
			ParentID: compressedTuple.ParentID,
		}
		tuples = append(tuples, tuple)
	}

	// Sort by time
	for i := 0; i < len(tuples)-1; i++ {
		for j := i + 1; j < len(tuples); j++ {
			if tuples[i].Snapshot.CreatedAt.Before(tuples[j].Snapshot.CreatedAt) {
				tuples[i], tuples[j] = tuples[j], tuples[i]
			}
		}
	}

	// Apply the limit
	if limit > 0 && len(tuples) > limit {
		tuples = tuples[:limit]
	}

	return tuples, nil
}

func (c *CompressedCheckpointer) PutWrites(config map[string]interface{}, writes []map[string]interface{}, taskID string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	threadID, ok := config["thread_id"].(string)
	if !ok {
		return fmt.Errorf("thread_id not found in config")
	}

	key := fmt.Sprintf("%s:%s", threadID, taskID)
	c.writes[key] = writes
	return nil
}

// GetCompressionStats reports compression statistics
func (c *CompressedCheckpointer) GetCompressionStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var totalOriginalSize, totalCompressedSize int
	for _, checkpoint := range c.checkpoints {
		totalOriginalSize += checkpoint.Size
		totalCompressedSize += checkpoint.CompressedSize
	}

	compressionRatio := 0.0
	if totalOriginalSize > 0 {
		compressionRatio = float64(totalCompressedSize) / float64(totalOriginalSize)
	}

	return map[string]interface{}{
		"total_checkpoints":     len(c.checkpoints),
		"total_original_size":   totalOriginalSize,
		"total_compressed_size": totalCompressedSize,
		"compression_ratio":     compressionRatio,
		"space_saved":           totalOriginalSize - totalCompressedSize,
	}
}

// compressData compresses data
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompressData decompresses data
func decompressData(compressedData []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// MemoryThreadManager manages threads in memory
type MemoryThreadManager struct {
	threads map[string]*Thread
	mutex   sync.RWMutex
}

// NewMemoryThreadManager constructs a memory-backed thread manager
func NewMemoryThreadManager() *MemoryThreadManager {
	return &MemoryThreadManager{
		threads: make(map[string]*Thread),
	}
}

// CreateThread creates a new thread
func (m *MemoryThreadManager) CreateThread(metadata map[string]interface{}) (*Thread, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	threadID := uuid.New().String()
	thread := &Thread{
		ID:        threadID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  metadata,
	}

	m.threads[threadID] = thread
	return thread, nil
}

// GetThread fetches a thread
func (m *MemoryThreadManager) GetThread(threadID string) (*Thread, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	thread, exists := m.threads[threadID]
	if !exists {
		return nil, fmt.Errorf("thread %s not found", threadID)
	}

	return thread, nil
}

// UpdateThread updates a thread
func (m *MemoryThreadManager) UpdateThread(threadID string, metadata map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	thread, exists := m.threads[threadID]
	if !exists {
		return fmt.Errorf("thread %s not found", threadID)
	}

	thread.UpdatedAt = time.Now()
	if metadata != nil {
		thread.Metadata = metadata
	}

	return nil
}

// DeleteThread removes a thread
func (m *MemoryThreadManager) DeleteThread(threadID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.threads[threadID]; !exists {
		return fmt.Errorf("thread %s not found", threadID)
	}

	delete(m.threads, threadID)
	return nil
}

// ListThreads lists all threads
func (m *MemoryThreadManager) ListThreads() ([]*Thread, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	threads := make([]*Thread, 0, len(m.threads))
	for _, thread := range m.threads {
		threads = append(threads, thread)
	}

	return threads, nil
}

// MemoryStateSnapshotManager manages snapshots in memory
type MemoryStateSnapshotManager struct {
	snapshots       map[string]*StateSnapshot
	threadSnapshots map[string][]*StateSnapshot // threadID -> snapshots
	mutex           sync.RWMutex
}

// NewMemoryStateSnapshotManager constructs an in-memory snapshot manager
func NewMemoryStateSnapshotManager() *MemoryStateSnapshotManager {
	return &MemoryStateSnapshotManager{
		snapshots:       make(map[string]*StateSnapshot),
		threadSnapshots: make(map[string][]*StateSnapshot),
	}
}

// SaveSnapshot stores a snapshot
func (m *MemoryStateSnapshotManager) SaveSnapshot(snapshot *StateSnapshot) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if snapshot.ID == "" {
		snapshot.ID = uuid.New().String()
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now()
	}

	m.snapshots[snapshot.ID] = snapshot

	// Add to the thread's snapshot list
	if m.threadSnapshots[snapshot.ThreadID] == nil {
		m.threadSnapshots[snapshot.ThreadID] = make([]*StateSnapshot, 0)
	}
	m.threadSnapshots[snapshot.ThreadID] = append(m.threadSnapshots[snapshot.ThreadID], snapshot)

	return nil
}

// GetSnapshot retrieves a snapshot
func (m *MemoryStateSnapshotManager) GetSnapshot(snapshotID string) (*StateSnapshot, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	snapshot, exists := m.snapshots[snapshotID]
	if !exists {
		return nil, fmt.Errorf("snapshot %s not found", snapshotID)
	}

	return snapshot, nil
}

// GetLatestSnapshot retrieves the most recent snapshot
func (m *MemoryStateSnapshotManager) GetLatestSnapshot(threadID string) (*StateSnapshot, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	snapshots, exists := m.threadSnapshots[threadID]
	if !exists || len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots found for thread %s", threadID)
	}

	// Find the newest snapshot
	var latest *StateSnapshot
	for _, snapshot := range snapshots {
		if latest == nil || snapshot.CreatedAt.After(latest.CreatedAt) {
			latest = snapshot
		}
	}

	return latest, nil
}

// GetStateHistory returns the snapshot history
func (m *MemoryStateSnapshotManager) GetStateHistory(threadID string) ([]*StateSnapshot, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	snapshots, exists := m.threadSnapshots[threadID]
	if !exists {
		return []*StateSnapshot{}, nil
	}

	// Copy and sort by time
	result := make([]*StateSnapshot, len(snapshots))
	copy(result, snapshots)

	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].CreatedAt.After(result[j].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

// ForkFromSnapshot creates a branch from a snapshot
func (m *MemoryStateSnapshotManager) ForkFromSnapshot(snapshotID string, newThreadID string) (*StateSnapshot, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	sourceSnapshot, exists := m.snapshots[snapshotID]
	if !exists {
		return nil, fmt.Errorf("snapshot %s not found", snapshotID)
	}

	// Generate a new thread ID when none is supplied
	if newThreadID == "" {
		newThreadID = uuid.New().String()
	}

	// Copy the snapshot to the new thread
	newSnapshot := &StateSnapshot{
		ID:        uuid.New().String(),
		ThreadID:  newThreadID,
		Values:    copyMap(sourceSnapshot.Values),
		Config:    copyMap(sourceSnapshot.Config),
		Metadata:  copyMap(sourceSnapshot.Metadata),
		CreatedAt: time.Now(),
		ParentID:  snapshotID, // Record the parent snapshot
	}

	m.snapshots[newSnapshot.ID] = newSnapshot
	m.threadSnapshots[newThreadID] = []*StateSnapshot{newSnapshot}

	return newSnapshot, nil
}

// DeleteSnapshot removes a snapshot
func (m *MemoryStateSnapshotManager) DeleteSnapshot(snapshotID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	snapshot, exists := m.snapshots[snapshotID]
	if !exists {
		return fmt.Errorf("snapshot %s not found", snapshotID)
	}

	delete(m.snapshots, snapshotID)

	threadSnapshots := m.threadSnapshots[snapshot.ThreadID]
	for i, s := range threadSnapshots {
		if s.ID == snapshotID {
			m.threadSnapshots[snapshot.ThreadID] = append(threadSnapshots[:i], threadSnapshots[i+1:]...)
			break
		}
	}

	return nil
}
