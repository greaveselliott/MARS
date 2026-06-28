# Reference: AI Engineering System Design Reading List

**Source list:** Neo Kim LinkedIn post, "If you want to become good at AI engineering (in 3 weeks), then learn these 15 concepts"
**OP credit:** [Neo Kim](https://www.linkedin.com/in/neokim/), author of [The System Design Newsletter](https://newsletter.systemdesign.one/)
**Source verified:** 2026-05-24
**Type:** Reference reading list

## Summary

This reading list captures the AI engineering resources surfaced by Neo Kim's
LinkedIn post and preserves them as durable MARS reference material.
The list is useful because it clusters agent memory, context engineering,
tooling, evals, RAG, MCP, and multi-agent architecture into one operator-facing
learning path.

MARS should treat these links as external reference material, not as
binding operating doctrine. Durable Mars rules still live in this repository's
design docs, feature contracts, role packets, skills, tools, and generated
target guidance.

## MARS Relevance

- **Agent state and memory:** informs checkpointing, state versioning, durable
  run memory, and consistency precedence across chat, docs, tests, tools, git,
  and traces.
- **Context engineering:** reinforces the `Context Efficiency` tenet by
  favoring routed context, targeted retrieval, and source-backed prompts over
  prompt stuffing.
- **Tool and MCP design:** supports Mars' universal tool surface and the need
  for permissioned, auditable, model-neutral tool execution.
- **Eval and scoring:** supports role scorecards, golden tasks, LLM-as-judge
  caution, and release-quality evidence loops.
- **Multi-agent architecture:** provides vocabulary for orchestrator-worker,
  pipeline, hierarchical, and shared-state patterns while warning against
  premature agent splitting.

## Original Reading List

| # | Topic | Link | Credit |
| --- | --- | --- | --- |
| 1 | AI Agents: Memory, State & Consistency | [AI Agents: State, Memory, Consistency - A Deep Dive](https://newsletter.systemdesign.one/p/ai-agent-memory) | Neo Kim and Sivasankar Natarajan, The System Design Newsletter |
| 2 | Machine Learning System Design 101 | [LinkedIn short link, unresolved during capture](https://lnkd.in/dFGuMknJ) | Neo Kim original list item; final destination should be verified before citation in design docs |
| 3 | Design Personal AI Chat Assistant | [Design a personal AI chat assistant](https://newsletter.systemdesign.one/p/ai-chat-assistant) | Neo Kim and Louis-Francois Bouchard, The System Design Newsletter |
| 4 | How RAG Works | [RAG - A Deep Dive](https://newsletter.systemdesign.one/p/how-rag-works) | Neo Kim and Eric Roby, The System Design Newsletter |
| 5 | LLM Concepts - A Deep Dive | [LLM Concepts](https://newsletter.systemdesign.one/p/llm-concepts) | Neo Kim and Louis-Francois Bouchard, The System Design Newsletter |
| 6 | How to Design an AI Agent | [How Do AI Agents Work](https://newsletter.systemdesign.one/p/how-do-ai-agents-work) | Neo Kim and Fran Soto, The System Design Newsletter |
| 7 | What is Reinforcement Learning | [21 Reinforcement Learning Concepts Explained Simply](https://newsletter.systemdesign.one/p/what-is-reinforcement-learning) | Neo Kim and Dr. Ashish Bamania, The System Design Newsletter |
| 8 | How Vector Databases Work | [What is a Vector Database](https://newsletter.systemdesign.one/p/what-is-a-vector-database) | Maxine Meurer and Neo Kim, The System Design Newsletter |
| 9 | Context Engineering 101 | [Context Engineering 101: How ChatGPT Stays on Track](https://newsletter.systemdesign.one/p/what-is-context-engineering) | Neo Kim and Louis-Francois Bouchard, The System Design Newsletter |
| 10 | AI Coding Workflow 101 | [I struggled to code with AI until I learned this workflow](https://newsletter.systemdesign.one/p/ai-coding-workflow) | Neo Kim and Louis-Francois Bouchard, The System Design Newsletter |
| 11 | LLM Evals Explained | [LLM Evals](https://newsletter.systemdesign.one/p/llm-evals) | Anshuman Mishra and Neo Kim, The System Design Newsletter |
| 12 | How AI Agents Work | [AI Agents Explained](https://newsletter.systemdesign.one/p/ai-agents-explained) | Sairam Sundaresan and Neo Kim, The System Design Newsletter |
| 13 | How MCP Works | [How MCP Works](https://newsletter.systemdesign.one/p/how-mcp-works) | Eric Roby and Neo Kim, The System Design Newsletter |
| 14 | Agentic Patterns Explained | [Agentic Design Patterns](https://newsletter.systemdesign.one/p/agentic-design-patterns) | Neo Kim, The System Design Newsletter |
| 15 | Multi-Agent Architecture Explained | [Multi-Agent Architectures, Clearly Explained](https://newsletter.systemdesign.one/p/multi-agent-system) | Neo Kim, The System Design Newsletter |

## Adoption Guidance

Use this list when updating or reviewing MARS design docs related to
agent runtime, context assembly, tool registry/MCP, scoring/evals, memory,
state checkpoints, and multi-agent orchestration. When a reference materially
changes Mars behavior, cite the specific article in the owning design doc and
translate the idea into Mars-native rules, tests, or tool behavior.
