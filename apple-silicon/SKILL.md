---
name: apple-silicon
description: "Apple Silicon optimization for VMs, ML inference, and compression. Covers Metal GPU acceleration, MLX model inference, CoreML conversion, M-series zstd compression, Apple Virtualization Framework, and performance tuning."
author: gastown
version: 1.0.0
---

# Apple Silicon Optimization Skill

Comprehensive guide for optimizing applications on Apple Silicon (M1/M2/M3/M4) including GPU acceleration, ML inference, virtualization, and compression.

## Overview

Apple Silicon's unified memory architecture and specialized hardware enable significant performance gains when properly optimized:

| Component | Capability | Use Case |
|-----------|------------|----------|
| **Neural Engine** | 16+ TOPS | Quantized model inference |
| **Metal GPU** | Up to 76 cores | Vector search, embeddings |
| **Unified Memory** | Up to 192GB | Large model loading |
| **ProRes Engine** | Hardware decode | Media processing |

## Metal GPU Acceleration

### Check Metal Availability

```typescript
import { metalAccelerator, isMetalAvailable, getMLCapabilities } from '@/lib/ml/metal-accelerator';

// Check if Metal is available
const available = await isMetalAvailable();

// Get device capabilities
const info = await getMLCapabilities();
console.log(`Device: ${info.name}`);
console.log(`Neural Engine: ${info.neuralEngineAvailable}`);
console.log(`Metal: ${info.metalAvailable}`);
console.log(`Max memory: ${info.recommendedMaxWorkingSetSize / 1024 / 1024 / 1024}GB`);
```

### GPU-Accelerated Vector Search

```typescript
import { metalAccelerator, search } from '@/lib/ml/metal-accelerator';

// Initialize accelerator
await metalAccelerator.initialize();

// Generate embedding
const queryEmbedding = await metalAccelerator.generateEmbedding("search query");

// GPU-accelerated similarity search (<10ms for 1K vectors)
const results = await search(queryEmbedding, vectorDatabase, 10);

// Batch search for multiple queries
const batchResults = await metalAccelerator.vectorSearchBatch(
  [query1, query2, query3],
  vectors,
  10
);
```

### Streaming Text Generation

```typescript
import { metalAccelerator } from '@/lib/ml/metal-accelerator';

// Stream tokens with Metal acceleration
for await (const token of metalAccelerator.generateText("Explain quantum computing", {
  model: 'mistral-7b-int8',
  maxTokens: 512,
  temperature: 0.7,
})) {
  process.stdout.write(token);
}
```

## MLX Model Inference

MLX is Apple's native ML framework optimized for unified memory architecture.

### Initialize MLX Provider

```typescript
import { mlxProvider, isMLXAvailable } from '@/lib/ml/mlx-provider';

// Check availability (Apple Silicon required)
const available = await isMLXAvailable();

// Initialize provider
await mlxProvider.initialize();

// Get system info
const info = await mlxProvider.getSystemInfo();
console.log(`Device: ${info.deviceName}`);
console.log(`Unified Memory: ${info.unifiedMemory / 1024 / 1024 / 1024}GB`);
console.log(`MLX Version: ${info.mlxVersion}`);
```

### Load and Run Models

```typescript
import { mlxProvider } from '@/lib/ml/mlx-provider';

// Load a quantized model
await mlxProvider.loadModel({
  name: 'mlx-community/Mistral-7B-Instruct-v0.3-4bit',
  quantization: 'int4',
  maxContextLength: 4096,
  device: 'auto'
});

// Generate text
const result = await mlxProvider.generateText("Write a haiku about coding", {
  maxTokens: 100,
  temperature: 0.8,
});

console.log(result.text);
console.log(`${result.metrics.tokensPerSecond} tokens/sec`);
```

### Streaming Generation

```typescript
import { mlxProvider } from '@/lib/ml/mlx-provider';

// Stream with metrics
for await (const token of mlxProvider.generateTextStream(prompt, options)) {
  process.stdout.write(token.token);
}
```

### Embedding Generation

```typescript
import { mlxProvider, generateEmbedding, generateEmbeddingsBatch } from '@/lib/ml/mlx-provider';

// Single embedding
const embedding = await generateEmbedding("Hello world");

// Batch embeddings (more efficient)
const embeddings = await generateEmbeddingsBatch([
  "First document",
  "Second document",
  "Third document"
], { batchSize: 32 });

// RAG-optimized embeddings
const queryEmb = await mlxProvider.generateRAGEmbedding("search query", true);
const docEmb = await mlxProvider.generateRAGEmbedding("document text", false);
```

### RAG Answer Generation

```typescript
import { mlxProvider } from '@/lib/ml/mlx-provider';

const answer = await mlxProvider.generateRAGAnswer(
  "What is the capital of France?",
  [
    "Paris is the capital and largest city of France.",
    "France is a country in Western Europe."
  ],
  { temperature: 0.3 }
);
```

## CoreML Model Conversion

Convert models from PyTorch/TensorFlow to CoreML for optimized on-device inference.

### Install coremltools

```bash
# Stable release
pip install coremltools

# Beta with LLM features (macOS 15+)
pip install coremltools==9.0b1
```

### Convert PyTorch Model

```python
import coremltools as ct
import torch

# Load and trace model
model = YourModel()
model.eval()
example_input = torch.randn(1, 3, 224, 224)
traced_model = torch.jit.trace(model, example_input)

# Convert to CoreML
mlmodel = ct.convert(
    traced_model,
    inputs=[ct.TensorType(shape=example_input.shape)],
    minimum_deployment_target=ct.target.iOS17
)

# Save
mlmodel.save("model.mlpackage")
```

### Quantize for Performance

```python
from coremltools.optimize.coreml import linear_quantize_weights, palettize_weights

# INT4 quantization (4x compression)
quantized = linear_quantize_weights(mlmodel, mode="linear_symmetric", dtype="int4")

# Palettization (weight clustering)
palettized = palettize_weights(mlmodel, nbits=4)
```

### Convert Embedding Models

```python
import coremltools as ct
import torch
import numpy as np
from sentence_transformers import SentenceTransformer

# Load model
model = SentenceTransformer('all-MiniLM-L6-v2')

class EmbeddingWrapper(torch.nn.Module):
    def __init__(self, model):
        super().__init__()
        self.model = model

    def forward(self, input_ids, attention_mask):
        outputs = self.model({'input_ids': input_ids, 'attention_mask': attention_mask})
        return outputs['sentence_embedding']

wrapper = EmbeddingWrapper(model)
wrapper.eval()

# Trace and convert
traced = torch.jit.trace(wrapper, (
    torch.randint(0, 30522, (1, 128)),
    torch.ones(1, 128, dtype=torch.long)
))

mlmodel = ct.convert(
    traced,
    inputs=[
        ct.TensorType(name="input_ids", shape=(1, 128), dtype=np.int32),
        ct.TensorType(name="attention_mask", shape=(1, 128), dtype=np.int32)
    ],
    minimum_deployment_target=ct.target.iOS16
)
```

### LLM Conversion with KV-Cache (macOS 15+)

```python
import coremltools as ct
import numpy as np

# Convert with stateful KV-cache for fast inference
mlmodel = ct.convert(
    traced_llm,
    inputs=[ct.TensorType(shape=(1, seq_len), dtype=np.int32)],
    states=[
        ct.StateType(
            wrapped_type=ct.TensorType(shape=(1, 32, 2048, 128)),
            name="kv_cache"
        )
    ],
    minimum_deployment_target=ct.target.macOS15
)
```

## M-Series zstd Compression

Apple Silicon's hardware acceleration benefits zstd compression.

### Optimal Settings for M-Series

```bash
# Fast compression (real-time, ~500 MB/s)
zstd -1 -T0 file.tar

# Balanced (default, ~200 MB/s)
zstd -3 -T0 file.tar

# High compression (~50 MB/s, best ratio)
zstd -19 -T0 file.tar

# Ultra compression with long range matching
zstd --ultra -22 --long=31 -T0 file.tar
```

### Programmatic Usage

```typescript
import { spawn } from 'child_process';

async function compressWithZstd(input: string, output: string, level = 3): Promise<void> {
  return new Promise((resolve, reject) => {
    const zstd = spawn('zstd', [
      `-${level}`,
      '-T0',          // Use all CPU cores
      '--rm',         // Remove input after compression
      '-o', output,
      input
    ]);
    zstd.on('close', (code) => code === 0 ? resolve() : reject(new Error(`zstd failed: ${code}`)));
  });
}

// Compress VM image
await compressWithZstd('alpine.raw', 'alpine.raw.zst', 19);
```

### Performance Benchmarks (M1 Max)

| Level | Speed | Ratio | Use Case |
|-------|-------|-------|----------|
| -1 | 500 MB/s | 2.5x | Real-time streaming |
| -3 | 200 MB/s | 3.0x | General purpose |
| -9 | 80 MB/s | 3.3x | Distribution |
| -19 | 15 MB/s | 3.5x | Archive |
| -22 --ultra | 5 MB/s | 3.7x | Maximum compression |

## Apple Virtualization Framework

Native VM support using Apple's Virtualization.framework.

### Basic VM Configuration (Swift)

```swift
import Virtualization

// Create VM configuration
let config = VZVirtualMachineConfiguration()

// CPU and memory
config.cpuCount = 4
config.memorySize = 4 * 1024 * 1024 * 1024  // 4GB

// Boot loader (Linux)
let bootLoader = VZLinuxBootLoader(kernelURL: kernelURL)
bootLoader.commandLine = "console=hvc0 root=/dev/vda"
config.bootLoader = bootLoader

// Storage
let diskAttachment = try VZDiskImageStorageDeviceAttachment(
    url: diskURL,
    readOnly: false
)
config.storageDevices = [VZVirtioBlockDeviceConfiguration(attachment: diskAttachment)]

// Network
let networkAttachment = VZNATNetworkDeviceAttachment()
config.networkDevices = [VZVirtioNetworkDeviceConfiguration(attachment: networkAttachment)]

// Create and start VM
let vm = VZVirtualMachine(configuration: config)
try await vm.start()
```

### Apple Container Runtime (macOS 26+)

```bash
# Install Apple Container CLI
curl -L -o container.pkg \
  "https://github.com/apple/container/releases/download/0.9.0/container-0.9.0-installer-signed.pkg"
sudo installer -pkg container.pkg -target /
container system start

# Run container (VM-per-container isolation)
container run -d --name web -p 8080:80 nginx:latest

# Each container gets dedicated IP (no port forwarding needed)
container inspect web --format '{{.NetworkSettings.IPAddress}}'

# Resource limits
container run --memory 2g --cpus 2 myapp:latest

# Build OCI-compatible images
container build -t myapp:latest .

# Push to any OCI registry
container push myapp:latest ghcr.io/org/myapp:latest
```

### Lima/vfkit VM Management

```bash
# Create Lima VM with VZ driver
limactl create --vm-type=vz --cpus=4 --memory=8 --name=dev

# Start VM
limactl start dev

# Shell into VM
limactl shell dev

# Mount host directory
limactl create --mount-writable --mount=/Users:/Users dev
```

## Performance Tuning

### Memory Optimization

```typescript
// Monitor MLX memory usage
const memory = await mlxProvider.getMemoryUsage();
console.log(`Used: ${memory.used / 1024 / 1024}MB`);
console.log(`Model: ${memory.modelMemory / 1024 / 1024}MB`);
console.log(`Cache: ${memory.cacheMemory / 1024 / 1024}MB`);

// Clear cache when needed
await mlxProvider.clearCache();
```

### Model Selection by Device

| Device | Recommended Models | Max Model Size |
|--------|-------------------|----------------|
| M1 (8GB) | 7B-int4, embeddings | ~4GB |
| M1 Pro (16GB) | 7B-int8, 13B-int4 | ~10GB |
| M1 Max (32GB) | 13B-int8, 30B-int4 | ~24GB |
| M1 Ultra (64GB) | 30B-int8, 70B-int4 | ~48GB |
| M2/M3/M4 variants | Same + 10-20% faster | Same |

### Inference Performance Targets

| Model | Device | Target Speed |
|-------|--------|--------------|
| Embedding (384d) | M1 | <50ms |
| 7B-int4 | M1 Max | 30-40 tok/s |
| 7B-int8 | M1 Max | 20-25 tok/s |
| 13B-int4 | M1 Ultra | 25-30 tok/s |
| 70B-int4 | M1 Ultra | 10-15 tok/s |

### Power Management

```bash
# Check thermal state
sudo powermetrics --samplers smc -n 1

# Monitor GPU usage
sudo powermetrics --samplers gpu_power -n 1

# Check ANE usage (Neural Engine)
sudo powermetrics --samplers ane_power -n 1
```

## Datadog Integration

Track MLX operations with distributed tracing.

```typescript
import { mlxTracedProvider, traceMLXOperation } from '@/lib/ml/mlx-ddtrace';

// Use traced provider
const result = await mlxTracedProvider.generateText(prompt, options, {
  tags: { 'user.id': userId },
  sessionId: sessionId
});

// Custom traced operation
await traceMLXOperation('custom_inference', async () => {
  // Your ML code
}, { 'model.name': 'custom-model' });
```

## Best Practices

### Do

1. **Use INT4 quantization** for LLMs - 4x smaller with <5% quality loss
2. **Batch embeddings** - significantly faster than one-by-one
3. **Enable unified memory** - let MLX manage memory allocation
4. **Use streaming** - better UX for text generation
5. **Monitor memory** - prevent OOM with large models

### Avoid

1. **FP32 models** - waste memory, no accuracy benefit
2. **Small batch sizes** - underutilize GPU
3. **Excessive model switching** - loading is expensive
4. **Ignoring thermal throttling** - sustained loads may slow down

## Files

| Path | Description |
|------|-------------|
| `src/lib/ml/metal-accelerator.ts` | Metal GPU acceleration API |
| `src/lib/ml/mlx-provider.ts` | MLX model inference |
| `src/lib/ml/mlx-ddtrace.ts` | Datadog tracing for MLX |
| `docs/research/apple-coremltools-integration.md` | CoreML conversion research |
| `docs/research/apple-containerization-integration.md` | Containerization research |
| `docs/research/apple-container-runtime-integration.md` | Container runtime research |

## References

- [MLX Framework](https://github.com/ml-explore/mlx)
- [Apple coremltools](https://github.com/apple/coremltools)
- [Apple Containerization](https://github.com/apple/containerization)
- [Apple Container CLI](https://github.com/apple/container)
- [Virtualization Framework](https://developer.apple.com/documentation/virtualization)
- [Metal Performance Shaders](https://developer.apple.com/documentation/metalperformanceshaders)
