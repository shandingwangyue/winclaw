---
name: news
description: Fetches latest news stories from Hacker News. Use when user wants to read tech news, top stories, or ask "what's new".
---

# News Skill

Fetches latest news from Hacker News (free, no API key required).

## Parameters

- `category` (string, optional): News category - "top", "best", or "ask"
- `limit` (number, optional): Number of stories to fetch (default: 10, max: 30)

## Returns

List of news items with title, URL, score, author, and comment count.

## Example

```json
{"category": "top", "limit": 10}
{"category": "best", "limit": 5}
{"category": "ask", "limit": 10}
```
