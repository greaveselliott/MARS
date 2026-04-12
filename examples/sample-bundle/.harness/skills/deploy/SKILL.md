---
name: deploy
description: How to deploy this project.
scope: all
---

# Deploy

This is a sample skill that teaches mars-harness agents how to deploy this project.

## Steps

1. Run tests: `npm test`
2. Build: `npm run build`
3. Deploy to staging: `npm run deploy:staging`
4. Verify health: `curl https://staging.example.com/healthz`
5. If healthy, promote to production: `npm run deploy:prod`

## Rollback

If the health check fails after deploy:

```bash
npm run rollback
```
