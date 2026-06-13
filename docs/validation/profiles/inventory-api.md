---
slug: inventory-api
archetype: api-service
title: Inventory API Demo
---

# Inventory API Demo

Build a small standard-library Go HTTP JSON API for managing inventory items.

The service should run locally, expose JSON endpoints to create inventory items, list items, fetch a single item, and update an item's stock quantity. Each item has a name, a quantity, and a reorder threshold. The API should report which items are at or below their reorder threshold, validate bad input with clear JSON error responses, and include a health endpoint.
