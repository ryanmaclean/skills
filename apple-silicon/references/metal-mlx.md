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

