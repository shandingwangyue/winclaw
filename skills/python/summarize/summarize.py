#!/usr/bin/env python3
"""
Summarize Skill for WinClaw
Text summarization using extractive method
"""

import sys
import json
import re


def summarize_text(text, max_sentences=3):
    sentences = re.split(r"(?<=[.!?])\s+", text)
    sentences = [s.strip() for s in sentences if s.strip()]

    if len(sentences) <= max_sentences:
        return text

    word_freq = {}
    for sentence in sentences:
        words = re.findall(r"\b\w+\b", sentence.lower())
        for word in words:
            if len(word) > 3:
                word_freq[word] = word_freq.get(word, 0) + 1

    scored_sentences = []
    for i, sentence in enumerate(sentences):
        words = re.findall(r"\b\w+\b", sentence.lower())
        score = sum(word_freq.get(w, 0) for w in words)
        score += (len(sentences) - i) * 0.1
        scored_sentences.append((score, i, sentence))

    scored_sentences.sort(key=lambda x: x[0], reverse=True)
    selected = sorted(scored_sentences[:max_sentences], key=lambda x: x[1])

    return " ".join(s[2] for s in selected)


def count_words(text):
    words = re.findall(r"\b\w+\b", text)
    return len(words)


def extract_keywords(text, top_n=5):
    words = re.findall(r"\b\w+\b", text.lower())
    stopwords = {
        "the",
        "a",
        "an",
        "and",
        "or",
        "but",
        "is",
        "are",
        "was",
        "were",
        "be",
        "been",
        "being",
        "have",
        "has",
        "had",
        "do",
        "does",
        "did",
        "will",
        "would",
        "could",
        "should",
        "may",
        "might",
        "must",
        "shall",
        "to",
        "of",
        "in",
        "for",
        "on",
        "with",
        "at",
        "by",
        "from",
        "as",
        "into",
        "through",
        "during",
        "before",
        "after",
        "above",
        "below",
        "that",
        "this",
        "these",
        "those",
        "it",
        "its",
        "they",
        "them",
        "their",
        "what",
        "which",
        "who",
        "whom",
        "where",
        "when",
        "why",
        "how",
    }

    filtered = [w for w in words if w not in stopwords and len(w) > 3]

    freq = {}
    for word in filtered:
        freq[word] = freq.get(word, 0) + 1

    sorted_words = sorted(freq.items(), key=lambda x: x[1], reverse=True)
    return [w[0] for w in sorted_words[:top_n]]


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No parameters provided"}))
        return

    try:
        params = json.loads(sys.argv[1])
    except:
        params = {}

    text = params.get("text", "")
    if not text:
        print(json.dumps({"error": "No text provided"}))
        return

    action = params.get("action", "summarize")
    max_sentences = params.get("max_sentences", 3)

    if action == "summarize":
        summary = summarize_text(text, max_sentences)
        print(
            json.dumps(
                {
                    "summary": summary,
                    "original_length": count_words(text),
                    "summary_length": count_words(summary),
                }
            )
        )
    elif action == "keywords":
        keywords = extract_keywords(text, params.get("top_n", 5))
        print(json.dumps({"keywords": keywords}))
    elif action == "stats":
        print(
            json.dumps(
                {
                    "word_count": count_words(text),
                    "char_count": len(text),
                    "sentence.split_count": len(re(r"[.!?]+", text)),
                }
            )
        )
    else:
        print(json.dumps({"error": f"Unknown action: {action}"}))


if __name__ == "__main__":
    main()
