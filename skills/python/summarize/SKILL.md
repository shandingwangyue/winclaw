---
name: summarize
description: Summarizes long text into shorter summaries and extracts keywords. Use when user wants to shorten text, extract key points, or get text statistics.
---

# Summarize Skill

Text summarization and keyword extraction using extractive algorithm.

## Parameters

- `text` (string, required): Text to summarize
- `action` (string, optional): Action to perform - "summarize", "keywords", or "stats"
- `max_sentences` (number, optional): Maximum sentences in summary (default: 3)
- `top_n` (number, optional): Number of keywords to extract (default: 5)

## Returns

Summarized text, word count, keywords, or text statistics.

## Example

```json
{"text": "Long article text...", "max_sentences": 3}
{"text": "Some text", "action": "keywords"}
{"text": "Some text", "action": "stats"}
```
