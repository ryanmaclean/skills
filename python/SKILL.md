# Skill: Python Development

## Overview

Python is a high-level, general-purpose programming language widely used for scripting, automation, data engineering, machine learning, and web development. This skill covers project setup, dependency management, testing, and common patterns.

## Key Concepts

- **Virtual environment**: An isolated Python environment with its own packages.
- **Package manager**: `pip` (standard) or `uv` / `poetry` for faster/richer dependency management.
- **Module / Package**: A `.py` file or directory containing Python code.
- **Type hints**: Optional annotations for parameters and return values (`def fn(x: int) -> str:`).
- **Linter / Formatter**: Tools like `ruff`, `black`, or `flake8` that enforce code style.

## Common Tasks

### Create and activate a virtual environment
```bash
python3 -m venv .venv
source .venv/bin/activate       # Linux / macOS
.venv\Scripts\activate          # Windows
```

### Install dependencies
```bash
pip install -r requirements.txt
# or with uv (faster)
uv pip install -r requirements.txt
```

### Run a script
```bash
python script.py
python -m module_name
```

### Run tests with pytest
```bash
pip install pytest
pytest tests/
pytest -v --tb=short            # verbose with short tracebacks
```

### Format and lint code
```bash
pip install ruff
ruff check .                    # lint
ruff format .                   # format
```

### Build a package
```bash
pip install build
python -m build                 # creates dist/*.whl and dist/*.tar.gz
```

## Common Patterns

### Reading and writing files
```python
with open("data.txt", "r", encoding="utf-8") as f:
    contents = f.read()

with open("output.txt", "w", encoding="utf-8") as f:
    f.write("hello\n")
```

### Working with environment variables
```python
import os
value = os.environ.get("MY_VAR", "default_value")
```

### Simple HTTP request
```python
import urllib.request
import json

with urllib.request.urlopen("https://api.example.com/data") as response:
    data = json.loads(response.read())
```

### Dataclasses for structured data
```python
from dataclasses import dataclass, field
from typing import List

@dataclass
class Config:
    host: str = "localhost"
    port: int = 8080
    tags: List[str] = field(default_factory=list)
```

## Best Practices

- Pin dependency versions in `requirements.txt` or `pyproject.toml` for reproducible builds.
- Use type hints; run `mypy` or `pyright` to catch type errors early.
- Write tests alongside your code; aim for meaningful coverage of business logic.
- Follow [PEP 8](https://peps.python.org/pep-0008/) style guidelines.
- Keep secrets out of source code; use environment variables or secret managers.
- Use `logging` instead of `print` for production code.

## References

- [Python Official Docs](https://docs.python.org/3/)
- [Real Python](https://realpython.com/)
- [Ruff Linter](https://docs.astral.sh/ruff/)
- [pytest Docs](https://docs.pytest.org/)
