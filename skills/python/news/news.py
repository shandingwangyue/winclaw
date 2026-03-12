#!/usr/bin/env python3
"""
News Skill for WinClaw
Uses Hacker News API (free, no API key required)
"""

import sys
import json
import urllib.request


def get_top_stories(limit=10):
    try:
        url = "https://hacker-news.firebaseio.com/v0/topstories.json"
        with urllib.request.urlopen(url, timeout=10) as response:
            story_ids = json.loads(response.read().decode())

        stories = []
        for story_id in story_ids[:limit]:
            story_url = f"https://hacker-news.firebaseio.com/v0/item/{story_id}.json"
            with urllib.request.urlopen(story_url, timeout=10) as response:
                story = json.loads(response.read().decode())

            stories.append(
                {
                    "title": story.get("title", ""),
                    "url": story.get("url", ""),
                    "score": story.get("score", 0),
                    "by": story.get("by", ""),
                    "time": story.get("time", 0),
                    "comments": story.get("descendants", 0),
                }
            )

        return stories
    except Exception as e:
        return {"error": str(e)}


def get_best_stories(limit=10):
    try:
        url = "https://hacker-news.firebaseio.com/v0/beststories.json"
        with urllib.request.urlopen(url, timeout=10) as response:
            story_ids = json.loads(response.read().decode())

        stories = []
        for story_id in story_ids[:limit]:
            story_url = f"https://hacker-news.firebaseio.com/v0/item/{story_id}.json"
            with urllib.request.urlopen(story_url, timeout=10) as response:
                story = json.loads(response.read().decode())

            stories.append(
                {
                    "title": story.get("title", ""),
                    "url": story.get("url", ""),
                    "score": story.get("score", 0),
                    "by": story.get("by", ""),
                    "time": story.get("time", 0),
                    "comments": story.get("descendants", 0),
                }
            )

        return stories
    except Exception as e:
        return {"error": str(e)}


def get_ask_stories(limit=10):
    try:
        url = "https://hacker-news.firebaseio.com/v0/askstories.json"
        with urllib.request.urlopen(url, timeout=10) as response:
            story_ids = json.loads(response.read().decode())

        stories = []
        for story_id in story_ids[:limit]:
            story_url = f"https://hacker-news.firebaseio.com/v0/item/{story_id}.json"
            with urllib.request.urlopen(story_url, timeout=10) as response:
                story = json.loads(response.read().decode())

            stories.append(
                {
                    "title": story.get("title", ""),
                    "text": story.get("text", ""),
                    "by": story.get("by", ""),
                    "time": story.get("time", 0),
                    "comments": story.get("descendants", 0),
                }
            )

        return stories
    except Exception as e:
        return {"error": str(e)}


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No parameters provided"}))
        return

    try:
        params = json.loads(sys.argv[1])
    except:
        params = {}

    category = params.get("category", "top")
    limit = params.get("limit", 10)
    limit = min(limit, 30)

    if category == "top":
        news = get_top_stories(limit)
    elif category == "best":
        news = get_best_stories(limit)
    elif category == "ask":
        news = get_ask_stories(limit)
    else:
        news = get_top_stories(limit)

    print(json.dumps(news))


if __name__ == "__main__":
    main()
