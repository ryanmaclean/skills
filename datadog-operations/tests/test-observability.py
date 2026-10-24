#!/usr/bin/env python3
"""Test DD observability functionality"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent / "python" / "lib"))

from dd_observability import init_observability, finalize_observability

def test_basic_observability():
    """Test basic observability features"""
    print("[TEST] Testing DD observability...")

    # Initialize
    obs = init_observability("test-script")
    print("✓ Observability initialized")

    # Test logging
    obs.log_info("Test info message")
    obs.log_warning("Test warning message")
    print("✓ Logging works")

    # Test spans
    with obs.span("test_operation"):
        obs.log_info("Inside test operation span")
    print("✓ Span context manager works")

    # Test metrics
    obs.gauge("test.metric", 42.0, tags=["env:test"])
    obs.count("test.count", 1, tags=["env:test"])
    print("✓ Metrics work")

    # Test API call recording
    obs.record_api_call("/api/test", "GET", 200, 150.5)
    print("✓ API call recording works")

    # Test result recording
    obs.record_result("test_key", "test_value")
    print("✓ Result recording works")

    # Finalize
    finalize_observability(0)
    print("✓ Observability finalized")

    print("\n✅ All observability tests passed!")

if __name__ == "__main__":
    test_basic_observability()
