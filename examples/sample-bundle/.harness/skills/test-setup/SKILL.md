---
name: test-setup
description: How to set up and run the test environment for this project.
scope: qa
---

# Test Environment Setup

## Prerequisites

- Node.js 20+
- Docker (for integration tests)

## Unit Tests

```bash
npm test
```

## Integration Tests

```bash
docker compose up -d
npm run test:integration
docker compose down
```

## Coverage

```bash
npm run test:coverage
```

Minimum threshold: 80% line coverage.
