"""
Test GLM-OCR (0.9B) via Ollama (local, free, no API key needed).

Setup:
  1. Install Ollama: brew install ollama
  2. Start server:   ollama serve  (if not already running)
  3. Pull model:     ollama pull glm-ocr
  4. Run:            python test_glm.py [image_path]

Alternative — Z.AI cloud API:
  1. Get API key: https://z.ai
  2. Export: export ZAI_API_KEY=...
  3. Run:   python test_glm.py [image_path] --cloud

Default test image: test_image.png
"""

import base64
import json
import os
import sys
import time
from urllib.request import Request, urlopen
from urllib.error import URLError

OLLAMA_URL = "http://localhost:11434/api/chat"
ZAI_URL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"

PROMPTS = {
    "extract": "请识别图片中的所有中文汉字，逐字列出。只输出汉字，不要拼音和翻译。",
    "structured": (
        "请识别图片中的所有中文文本。以JSON数组格式输出，每个元素包含："
        '{"text": "识别的文字", "type": "printed或handwritten"}。'
        "只输出JSON，不要其他内容。"
    ),
}


def encode_image(image_path: str) -> str:
    with open(image_path, "rb") as f:
        return base64.b64encode(f.read()).decode()


def call_ollama(image_path: str, prompt_key: str = "extract") -> dict:
    image_b64 = encode_image(image_path)
    prompt = PROMPTS.get(prompt_key, PROMPTS["extract"])

    payload = json.dumps({
        "model": "glm-ocr",
        "messages": [{"role": "user", "content": prompt, "images": [image_b64]}],
        "stream": False,
    }).encode()

    req = Request(OLLAMA_URL, data=payload, method="POST", headers={
        "Content-Type": "application/json",
    })

    print(f"Calling Ollama GLM-OCR ({prompt_key} mode)...")
    start = time.time()

    try:
        with urlopen(req, timeout=120) as resp:
            result = json.loads(resp.read().decode())
    except URLError as e:
        print(f"ERROR: Cannot connect to Ollama at {OLLAMA_URL}")
        print("  Make sure Ollama is running: ollama serve")
        print("  And model is pulled: ollama pull glm-ocr")
        print(f"  Error: {e}")
        sys.exit(1)

    elapsed = time.time() - start
    return {"result": result, "elapsed": elapsed}


def call_zai(image_path: str, prompt_key: str = "extract") -> dict:
    api_key = os.environ.get("ZAI_API_KEY", "")
    if not api_key:
        print("ERROR: Set ZAI_API_KEY environment variable")
        print("  export ZAI_API_KEY=...")
        sys.exit(1)

    image_b64 = encode_image(image_path)
    ext = image_path.rsplit(".", 1)[-1].lower()
    mime_map = {"png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg", "webp": "image/webp"}
    mime = mime_map.get(ext, "image/png")
    prompt = PROMPTS.get(prompt_key, PROMPTS["extract"])

    payload = json.dumps({
        "model": "glm-4v-flash",
        "messages": [{
            "role": "user",
            "content": [
                {"type": "image_url", "image_url": {"url": f"data:{mime};base64,{image_b64}"}},
                {"type": "text", "text": prompt},
            ],
        }],
        "max_tokens": 2048,
    }).encode()

    req = Request(ZAI_URL, data=payload, method="POST", headers={
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
    })

    print(f"Calling Z.AI GLM-4V-Flash ({prompt_key} mode)...")
    start = time.time()

    with urlopen(req, timeout=60) as resp:
        result = json.loads(resp.read().decode())

    elapsed = time.time() - start
    return {"result": result, "elapsed": elapsed}


def main():
    use_cloud = "--cloud" in sys.argv
    args = [a for a in sys.argv[1:] if a != "--cloud"]
    image_path = args[0] if args else "test_image.png"

    if not os.path.exists(image_path):
        print(f"ERROR: Image not found: {image_path}")
        sys.exit(1)

    print(f"Image:  {image_path}")
    print(f"Size:   {os.path.getsize(image_path) / 1024:.1f} KB")
    print(f"Engine: {'Z.AI Cloud (GLM-4V-Flash)' if use_cloud else 'Ollama Local (GLM-OCR 0.9B)'}")
    print("=" * 60)

    call_fn = call_zai if use_cloud else call_ollama

    # Test 1: Simple extraction
    resp = call_fn(image_path, "extract")

    if use_cloud:
        content = resp["result"]["choices"][0]["message"]["content"]
        usage = resp["result"].get("usage", {})
        token_info = f"Tokens: input={usage.get('prompt_tokens', '?')}, output={usage.get('completion_tokens', '?')}"
    else:
        content = resp["result"]["message"]["content"]
        token_info = (
            f"Tokens: input={resp['result'].get('prompt_eval_count', '?')}, "
            f"output={resp['result'].get('eval_count', '?')}"
        )

    print(f"\n--- Extract mode ({resp['elapsed']:.2f}s) ---")
    print(f"Output:\n{content}")
    print(f"\n{token_info}")

    # Test 2: Structured JSON
    print("\n" + "=" * 60)
    resp2 = call_fn(image_path, "structured")

    if use_cloud:
        content2 = resp2["result"]["choices"][0]["message"]["content"]
        usage2 = resp2["result"].get("usage", {})
        token_info2 = f"Tokens: input={usage2.get('prompt_tokens', '?')}, output={usage2.get('completion_tokens', '?')}"
    else:
        content2 = resp2["result"]["message"]["content"]
        token_info2 = (
            f"Tokens: input={resp2['result'].get('prompt_eval_count', '?')}, "
            f"output={resp2['result'].get('eval_count', '?')}"
        )

    print(f"\n--- Structured mode ({resp2['elapsed']:.2f}s) ---")
    print(f"Output:\n{content2}")
    print(f"\n{token_info2}")

    # Summary
    print("\n" + "=" * 60)
    print("SUMMARY:")
    print(f"  Engine:               {'Z.AI GLM-4V-Flash' if use_cloud else 'Ollama GLM-OCR 0.9B'}")
    print(f"  Latency (extract):    {resp['elapsed']:.2f}s")
    print(f"  Latency (structured): {resp2['elapsed']:.2f}s")
    print(f"  Per-char confidence:  NOT AVAILABLE (VLM generative output)")
    print(f"  Bounding boxes:       NOT AVAILABLE")


if __name__ == "__main__":
    main()
