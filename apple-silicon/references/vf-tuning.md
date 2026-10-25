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
