# Checkpoint & Recovery Examples

This directory contains comprehensive examples demonstrating the checkpoint and recovery capabilities of the MAS framework.

## Features Demonstrated

### 🔍 Basic Checkpoint Demo
- Automatic checkpoint saving during workflow execution
- File-based persistent storage
- Checkpoint compression and configuration
- Real-time checkpoint information display

### 💥 Failure Recovery Demo  
- Workflow failure simulation
- Automatic error checkpoint creation
- Recovery from specific checkpoint
- Continuation from interruption point

### 🎮 Interactive Checkpoint Management
- Interactive checkpoint operations
- Manual checkpoint creation
- Checkpoint listing and inspection
- Selective checkpoint deletion

### ⏱️ Long-Running Workflow Demo
- Time-based periodic checkpoints
- Progress monitoring during execution
- Interrupt and resume capabilities
- Multi-stage complex workflows

## Quick Start

```bash
# Set your API key
export OPENAI_API_KEY="your-api-key-here"

# Run the checkpoint demo
cd examples/checkpoint
go run main.go
```

## Usage Scenarios

### 1. Basic Checkpoint Workflow

```go
// Create checkpointer with file storage
checkpointer, err := mas.NewFileCheckpointer("./checkpoints",
    mas.WithAutoSave(true),
    mas.WithSaveAfterNode(true),
    mas.WithMaxCheckpoints(5),
    mas.WithCompression(true),
)

// Build workflow with checkpoint support
workflow := mas.NewWorkflow().
    WithCheckpointer(checkpointer).
    WithAutoSave(true).
    WithSaveAfterNode(true).
    AddNode(mas.NewAgentNode("researcher", researcher)).
    AddNode(mas.NewAgentNode("analyzer", analyzer)).
    AddEdge("researcher", "analyzer").
    SetStart("researcher")

// Execute with automatic checkpointing
result, err := workflow.ExecuteWithCheckpoint(ctx, initialData)
```

### 2. Recovery from Failure

```go
// First execution (may fail)
result, err := workflow.ExecuteWithCheckpointID(ctx, "my-workflow", data)
if err != nil {
    // Recovery from latest checkpoint
    result, err = workflow.ResumeFromCheckpoint(ctx, "my-workflow")
}
```

### 3. Manual Checkpoint Management

```go
// List all checkpoints for a workflow
checkpoints, err := checkpointer.List(ctx, workflowID)

// Resume from specific checkpoint
result, err := workflow.ResumeFromCheckpointID(ctx, workflowID, checkpointID)

// Clean up old checkpoints
err = checkpointer.DeleteAll(ctx, workflowID)
```

## Configuration Options

### Checkpoint Behavior
- `WithAutoSave(bool)`: Enable automatic checkpoint saving
- `WithSaveAfterNode(bool)`: Save checkpoint after each node
- `WithSaveBeforeNode(bool)`: Save checkpoint before each node
- `WithSaveInterval(duration)`: Periodic checkpoint interval
- `WithMaxCheckpoints(int)`: Maximum checkpoints to retain
- `WithCompression(bool)`: Enable checkpoint compression

### Storage Options
- `NewFileCheckpointer(path)`: File system storage
- `NewMemoryCheckpointer()`: In-memory storage (testing)
- Custom storage via `StateStore` interface

## Key Benefits

### 🛡️ Reliability
- **Fault Tolerance**: Automatically recover from failures
- **Progress Preservation**: Never lose completed work
- **Error Recovery**: Resume from last stable state

### 🚀 Performance  
- **Smart Resumption**: Skip already completed nodes
- **Efficient Storage**: Compressed checkpoint data
- **Minimal Overhead**: Lightweight checkpoint operations

### 🔧 Flexibility
- **Multiple Storage Backends**: File, memory, database
- **Configurable Policies**: Custom checkpoint strategies
- **Granular Control**: Fine-tune checkpoint behavior

## Advanced Features

### Error Handling
The system automatically creates checkpoints when errors occur, including:
- Node execution failures
- LLM API errors
- Tool execution errors
- Network interruptions

### Resume Logic
Smart resume capabilities include:
- **Node Skip**: Automatically skip completed nodes
- **State Restoration**: Restore full workflow context
- **Edge Traversal**: Correctly handle workflow routing
- **Conditional Logic**: Preserve conditional routing state

### Monitoring
Real-time monitoring features:
- **Progress Tracking**: Monitor workflow advancement
- **Checkpoint Status**: View checkpoint creation history
- **Storage Usage**: Track checkpoint storage consumption
- **Performance Metrics**: Measure checkpoint overhead

## Troubleshooting

### Common Issues

1. **"No checkpointer configured"**
   - Ensure you call `WithCheckpointer()` before execution
   - Verify checkpointer is properly initialized

2. **"Checkpoint not found"**
   - Check workflow ID spelling
   - Verify checkpoint storage location
   - Ensure checkpoints weren't deleted

3. **"Node not found during resume"**
   - Ensure workflow structure matches checkpoint
   - Verify all nodes are properly registered

### Debug Tips

- Enable verbose logging to see checkpoint operations
- Check checkpoint file structure in storage directory
- Validate checkpoint data integrity
- Monitor memory usage during long workflows

## Best Practices

### 📋 Development
- Use memory checkpointer for testing
- Enable compression for production
- Set reasonable checkpoint limits
- Test recovery scenarios

### 🏭 Production
- Use persistent storage (file/database)
- Configure appropriate retention policies
- Monitor checkpoint storage growth
- Implement cleanup procedures

### 🔒 Security
- Secure checkpoint storage location
- Consider encryption for sensitive workflows
- Implement access controls
- Audit checkpoint operations

## Next Steps

1. **Try the demos**: Run each demo scenario to understand capabilities
2. **Customize configuration**: Experiment with different checkpoint settings
3. **Build your workflow**: Apply checkpointing to your specific use cases
4. **Monitor performance**: Measure checkpoint impact on your workflows

For more information, see the main [MAS documentation](../../CLAUDE.md).