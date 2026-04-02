"""
Test DeepSeek-OCR-2 via API (OpenAI-compatible).

Setup:
  1. Get API key: https://platform.deepseek.com/api_keys
  2. Export: export DEEPSEEK_API_KEY=sk-...
  3. Run:    python test_deepseek.py [image_path]

Default test image: test_image.png
"""

import base64
import json
import os
import sys
import time
from urllib.request import Request, urlopen

API_KEY = os.environ.get("DEEPSEEK_API_KEY", "")
API_URL = "https://api.deepseek.com/chat/completions"

PROMPTS = {
    "extract": "请识别图片中的所有中文汉字，逐字列出。只输出汉字，不要拼音和翻译。",
    "structured": (
        "请识别图片中的所有中文文本。以JSON数组格式输出，每个元素包含："
        '{"text": "识别的文字", "type": "printed或handwritten"}。'
        "只输出JSON，不要其他内容。"
    ),
}


def encode_image(image_path: str) -> tuple[str, str]:
    ext = image_path.rsplit(".", 1)[-1].lower()
    mime_map = {"png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg", "webp": "image/webp"}
    mime = mime_map.get(ext, "image/png")
    with open(image_path, "rb") as f:
        return base64.b64encode(f.read()).decode(), mime


def call_deepseek(image_path: str, prompt_key: str = "extract") -> dict:
    if not API_KEY:
        print("ERROR: Set DEEPSEEK_API_KEY environment variable")
        print("  export DEEPSEEK_API_KEY=sk-...")
        sys.exit(1)

    image_b64, mime = encode_image(image_path)
    prompt = PROMPTS.get(prompt_key, PROMPTS["extract"])

    payload = json.dumps({
        "model": "deepseek-chat",
        "messages": [{
            "role": "user",
            "content": [
                {"type": "image_url", "image_url": {"url": f"data:{mime};base64,{image_b64}"}},
                {"type": "text", "text": prompt},
            ],
        }],
        "max_tokens": 2048,
    }).encode()

    req = Request(API_URL, data=payload, method="POST", headers={
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json",
    })

    print(f"Calling DeepSeek API ({prompt_key} mode)...")
    start = time.time()

    with urlopen(req) as resp:
        result = json.loads(resp.read().decode())

    elapsed = time.time() - start
    return {"result": result, "elapsed": elapsed}


def main():
    image_path = sys.argv[1] if len(sys.argv) > 1 else "test_image.png"
    if not os.path.exists(image_path):
        print(f"ERROR: Image not found: {image_path}")
        sys.exit(1)

    print(f"Image: {image_path}")
    print(f"Size:  {os.path.getsize(image_path) / 1024:.1f} KB")
    print("=" * 60)

    # Test 1: Simple extraction
    resp = call_deepseek(image_path, "extract")
    content = resp["result"]["choices"][0]["message"]["content"]
    usage = resp["result"].get("usage", {})

    print(f"\n--- Extract mode ({resp['elapsed']:.2f}s) ---")
    print(f"Output:\n{content}")
    print(f"\nTokens: input={usage.get('prompt_tokens', '?')}, output={usage.get('completion_tokens', '?')}")

    # Test 2: Structured JSON
    print("\n" + "=" * 60)
    resp2 = call_deepseek(image_path, "structured")
    content2 = resp2["result"]["choices"][0]["message"]["content"]
    usage2 = resp2["result"].get("usage", {})

    print(f"\n--- Structured mode ({resp2['elapsed']:.2f}s) ---")
    print(f"Output:\n{content2}")
    print(f"\nTokens: input={usage2.get('prompt_tokens', '?')}, output={usage2.get('completion_tokens', '?')}")

    # Summary
    print("\n" + "=" * 60)
    print("SUMMARY:")
    print(f"  Latency (extract):    {resp['elapsed']:.2f}s")
    print(f"  Latency (structured): {resp2['elapsed']:.2f}s")
    print(f"  Per-char confidence:  NOT AVAILABLE (VLM generative output)")
    print(f"  Bounding boxes:       NOT AVAILABLE")


if __name__ == "__main__":
    main()
