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

