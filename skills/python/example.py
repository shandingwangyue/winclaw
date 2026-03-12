#!/usr/bin/env python3
"""
Example Python Skill for WinClaw
"""

import sys
import json


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No parameters provided"}))
        return

    try:
        params = json.loads(sys.argv[1])
    except:
        params = {}

    action = params.get("action", "greet")

    if action == "greet":
        name = params.get("name", "User")
        result = f"Hello, {name}! This is a Python skill."
    elif action == "calculate":
        expr = params.get("expression", "0")
        try:
            result = str(eval(expr))
        except:
            result = "Error: Invalid expression"
    else:
        result = f"Unknown action: {action}"

    print(json.dumps({"result": result}))


if __name__ == "__main__":
    main()
