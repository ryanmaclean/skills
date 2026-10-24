---
description: "Query Real User Monitoring for frontend performance and user experience metrics"
argument-hint: "[APPLICATION] [--from TIMERANGE] [--view VIEW]"
---

# Datadog Real User Monitoring (RUM)

Query RUM to analyze frontend performance, user experience, browser metrics, and real user behavior.

## What is RUM?

Real User Monitoring tracks actual user experiences:
- **Page load performance** - Core Web Vitals, load times
- **User interactions** - Clicks, navigation, errors
- **Session replay** - Visual playback of user sessions
- **Frontend errors** - JavaScript errors, failed requests

**Official Documentation**: https://www.datadoghq.com/product/real-user-monitoring/

## Usage

```bash
# Query all RUM data
dd rum

# Query specific application
dd rum my-web-app

# Filter by view/page
dd rum my-app --view "/checkout"

# Time range
dd rum --from 24h
```

## Key Metrics

**Core Web Vitals**:
- LCP (Largest Contentful Paint)
- FID (First Input Delay)
- CLS (Cumulative Layout Shift)

**Performance**:
- Page load time
- Time to interactive
- First byte time
- Resource load times

**User Experience**:
- Error rate
- Session duration
- Bounce rate
- Frustration signals

## Why Use the CLI?

- **Fast access** - Check frontend performance instantly
- **Correlate with backend** - Link RUM to APM traces
- **Automate monitoring** - Include in deployment checks
- **Debug efficiently** - Quick error investigation

## Example Prompts

> "Show me RUM data for my web application"
> "What's the frontend error rate?"
> "Check Core Web Vitals scores"
> "Find slow page loads in the checkout flow"

## Learn More

- [RUM Product Page](https://www.datadoghq.com/product/real-user-monitoring/)