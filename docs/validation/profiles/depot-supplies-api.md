---
slug: depot-supplies-api
archetype: api-service
title: Depot Supplies API Demo
---

# Depot Supplies API Demo

Build a small standard-library Go HTTP JSON API for tracking consumable supplies in a maintenance depot.

The service should run locally and expose JSON endpoints to register a supply item, list all items, fetch one item by its item code, and record stock receipts and withdrawals (quantities received into or drawn from the depot) against an item. Each item has an item code, a name, a quantity on hand, and a reorder threshold. The API should report which items have dropped to or below their reorder threshold so the depot knows what to restock, reject invalid input with clear JSON error responses (unknown item code, negative quantities, malformed bodies), and include a health endpoint that reports service status.
