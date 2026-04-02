"""
Multi-engine OCR HTTP service.

Supported engines:
  - paddleocr (default) — PaddleOCR PP-OCRv5
  - tesseract           — Tesseract OCR (requires: brew install tesseract tesseract-lang && pip install pytesseract)

Input: JSON with base64-encoded image (same format as Google Vision / Baidu OCR APIs).

Post-processing: OCR returns text lines → jieba segments into words → filter CJK-only words.

Per-character confidence: Monkey-patches PaddleOCR's CTC postprocessor to return
per-character confidence scores instead of per-line averages.
See: https://github.com/PaddlePaddle/PaddleOCR/issues/5932
"""

import base64
import json
import os
import re
import tempfile
import time
from enum import Enum
from urllib.request import Request, urlopen
from urllib.error import URLError

os.environ["PADDLE_PDX_DISABLE_MODEL_SOURCE_CHECK"] = "True"

import numpy as np
import jieba
from pypinyin import pinyin, Style
from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from paddleocr import PaddleOCR
from pydantic import BaseModel

# ---------------------------------------------------------------------------
# Monkey-patch: return per-character confidence from CTC decoder
# ---------------------------------------------------------------------------
from paddlex.inference.models.text_recognition.processors import BaseRecLabelDecode, CTCLabelDecode
from paddlex.inference.models.text_recognition.predictor import TextRecPredictor


# --- Patch 1: BaseRecLabelDecode.decode → return (text, avg, char_scores) ---
def _patched_decode(self, text_index, text_prob=None, is_remove_duplicate=False, return_word_box=False):
    result_list = []
    ignored_tokens = self.get_ignored_tokens()
    batch_size = len(text_index)
    for batch_idx in range(batch_size):
        selection = np.ones(len(text_index[batch_idx]), dtype=bool)
        if is_remove_duplicate:
            selection[1:] = text_index[batch_idx][1:] != text_index[batch_idx][:-1]
        for ignored_token in ignored_tokens:
            selection &= text_index[batch_idx] != ignored_token

        char_list = [self.character[text_id] for text_id in text_index[batch_idx][selection]]
        if text_prob is not None:
            conf_list = text_prob[batch_idx][selection]
        else:
            conf_list = [1] * len(selection)
        if len(conf_list) == 0:
            conf_list = [0]

        text = "".join(char_list)
        if self.reverse:
            text = self.pred_reverse(text)

        avg_score = float(np.mean(conf_list))
        char_scores = [round(float(c), 4) for c in conf_list]

        if return_word_box:
            word_list, word_col_list, state_list = self.get_word_info(text, selection)
            result_list.append((text, avg_score, [len(text_index[batch_idx]), word_list, word_col_list, state_list], char_scores))
        else:
            result_list.append((text, avg_score, char_scores))
    return result_list


BaseRecLabelDecode.decode = _patched_decode


# --- Patch 2: CTCLabelDecode.__call__ → pass char_scores through ---
def _patched_ctc_call(self, pred, return_word_box=False, **kwargs):
    preds = np.array(pred[0])
    preds_idx = preds.argmax(axis=-1)
    preds_prob = preds.max(axis=-1)
    text = self.decode(preds_idx, preds_prob, is_remove_duplicate=True, return_word_box=return_word_box)
    if return_word_box:
        for rec_idx, rec in enumerate(text):
            wh_ratio = kwargs["wh_ratio_list"][rec_idx]
            max_wh_ratio = kwargs["max_wh_ratio"]
            rec[2][0] = rec[2][0] * (wh_ratio / max_wh_ratio)
    texts = []
    scores = []
    char_scores_list = []
    for t in text:
        if return_word_box:
            texts.append((t[0], t[2]))
            scores.append(t[1])
            char_scores_list.append(t[3])
        else:
            texts.append(t[0])
            scores.append(t[1])
            char_scores_list.append(t[2])
    return texts, scores, char_scores_list


CTCLabelDecode.__call__ = _patched_ctc_call


# --- Patch 3: TextRecPredictor.process → include rec_char_scores in result ---
_original_process = TextRecPredictor.process


def _patched_process(self, batch_data, return_word_box=False):
    batch_raw_imgs = self.pre_tfs["Read"](imgs=batch_data.instances)
    width_list = []
    for img in batch_raw_imgs:
        width_list.append(img.shape[1] / float(img.shape[0]))
    indices = np.argsort(np.array(width_list))
    batch_imgs = self.pre_tfs["ReisizeNorm"](imgs=batch_raw_imgs)
    x = self.pre_tfs["ToBatch"](imgs=batch_imgs)

    if self._use_static_model:
        batch_preds = self.infer(x=x)
    else:
        from paddlex.utils.device import TemporaryDeviceChanger
        with TemporaryDeviceChanger(self.device):
            batch_preds = self.infer(x=x)

    batch_num = self.batch_sampler.batch_size
    img_num = len(batch_raw_imgs)
    rec_image_shape = next(
        op["RecResizeImg"]["image_shape"]
        for op in self.config["PreProcess"]["transform_ops"]
        if "RecResizeImg" in op
    )
    imgC, imgH, imgW = rec_image_shape[:3]
    max_wh_ratio = imgW / imgH
    end_img_no = min(img_num, batch_num)
    wh_ratio_list = []
    for ino in range(0, end_img_no):
        h, w = batch_raw_imgs[indices[ino]].shape[0:2]
        wh_ratio = w * 1.0 / h
        max_wh_ratio = max(max_wh_ratio, wh_ratio)
        wh_ratio_list.append(wh_ratio)

    texts, scores, char_scores_list = self.post_op(
        batch_preds,
        return_word_box=return_word_box or self.return_word_box,
        wh_ratio_list=wh_ratio_list,
        max_wh_ratio=max_wh_ratio,
    )

    try:
        from bidi.algorithm import get_display
        if self.model_name in ("arabic_PP-OCRv3_mobile_rec", "arabic_PP-OCRv5_mobile_rec"):
            texts = [get_display(s) for s in texts]
    except ImportError:
        pass

    return {
        "input_path": batch_data.input_paths,
        "page_index": batch_data.page_indexes,
        "input_img": batch_raw_imgs,
        "rec_text": texts,
        "rec_score": scores,
        "rec_char_scores": char_scores_list,
        "vis_font": [self.vis_font] * len(batch_raw_imgs),
    }


TextRecPredictor.process = _patched_process


# --- Patch 4: OCR pipeline → collect rec_char_scores from rec model results ---
# The pipeline builds results dict but never collects rec_char_scores.
# We patch the predict generator to add rec_char_scores collection.
from paddlex.inference.pipelines.ocr.pipeline import _OCRPipeline, OCRResult
from paddlex.inference.pipelines.components import (
    CropByPolys, SortQuadBoxes, SortPolyBoxes, cal_ocr_word_box, convert_points_to_boxes, rotate_image,
)


def _patched_pipeline_predict(
    self, input, use_doc_orientation_classify=None, use_doc_unwarping=None,
    use_textline_orientation=None, text_det_limit_side_len=None,
    text_det_limit_type=None, text_det_max_side_limit=None, text_det_thresh=None,
    text_det_box_thresh=None, text_det_unclip_ratio=None, text_rec_score_thresh=None,
    return_word_box=None,
):
    """Patched predict that also collects rec_char_scores."""
    model_settings = self.get_model_settings(
        use_doc_orientation_classify, use_doc_unwarping, use_textline_orientation
    )
    if not self.check_model_settings_valid(model_settings):
        yield {"error": "the input params for model settings are invalid!"}

    text_det_params = self.get_text_det_params(
        text_det_limit_side_len, text_det_limit_type, text_det_max_side_limit,
        text_det_thresh, text_det_box_thresh, text_det_unclip_ratio,
    )
    if text_rec_score_thresh is None:
        text_rec_score_thresh = self.text_rec_score_thresh
    if return_word_box is None:
        return_word_box = self.return_word_box

    for _, batch_data in enumerate(self.batch_sampler(input)):
        image_arrays = self.img_reader(batch_data.instances)

        if model_settings["use_doc_preprocessor"]:
            doc_preprocessor_results = list(
                self.doc_preprocessor_pipeline(
                    image_arrays,
                    use_doc_orientation_classify=use_doc_orientation_classify,
                    use_doc_unwarping=use_doc_unwarping,
                )
            )
        else:
            doc_preprocessor_results = [{"output_img": arr} for arr in image_arrays]

        doc_preprocessor_images = [item["output_img"] for item in doc_preprocessor_results]
        det_results = list(self.text_det_model(doc_preprocessor_images, **text_det_params))
        dt_polys_list = [item["dt_polys"] for item in det_results]
        dt_polys_list = [self._sort_boxes(item) for item in dt_polys_list]

        results = [
            {
                "input_path": input_path,
                "page_index": page_index,
                "doc_preprocessor_res": doc_preprocessor_res,
                "dt_polys": dt_polys,
                "model_settings": model_settings,
                "text_det_params": text_det_params,
                "text_type": self.text_type,
                "text_rec_score_thresh": text_rec_score_thresh,
                "return_word_box": return_word_box,
                "rec_texts": [],
                "rec_scores": [],
                "rec_char_scores": [],
                "rec_polys": [],
                "vis_fonts": [],
            }
            for input_path, page_index, doc_preprocessor_res, dt_polys in zip(
                batch_data.input_paths, batch_data.page_indexes,
                doc_preprocessor_results, dt_polys_list,
            )
        ]

        if return_word_box:
            for res in results:
                res["text_word"] = []
                res["text_word_region"] = []

        indices = [idx for idx in range(len(doc_preprocessor_images)) if len(dt_polys_list[idx]) > 0]

        if indices:
            all_subs_of_imgs = []
            chunk_indices = [0]
            for idx in indices:
                all_subs_of_img = list(self._crop_by_polys(doc_preprocessor_images[idx], dt_polys_list[idx]))
                all_subs_of_imgs.extend(all_subs_of_img)
                chunk_indices.append(chunk_indices[-1] + len(all_subs_of_img))

            if model_settings["use_textline_orientation"]:
                angles = [
                    int(info["class_ids"][0])
                    for info in self.textline_orientation_model(all_subs_of_imgs)
                ]
                all_subs_of_imgs = self.rotate_image(all_subs_of_imgs, angles)
            else:
                angles = [-1] * len(all_subs_of_imgs)
            for i, idx in enumerate(indices):
                results[idx]["textline_orientation_angles"] = angles[chunk_indices[i]:chunk_indices[i + 1]]

            for i, idx in enumerate(indices):
                all_subs_of_img = all_subs_of_imgs[chunk_indices[i]:chunk_indices[i + 1]]
                res = results[idx]
                dt_polys = dt_polys_list[idx]
                sub_img_info_list = [
                    {"sub_img_id": img_id, "sub_img_ratio": sub_img.shape[1] / float(sub_img.shape[0])}
                    for img_id, sub_img in enumerate(all_subs_of_img)
                ]
                sorted_subs_info = sorted(sub_img_info_list, key=lambda x: x["sub_img_ratio"])
                sorted_subs_of_img = [all_subs_of_img[x["sub_img_id"]] for x in sorted_subs_info]

                for j, rec_res in enumerate(self.text_rec_model(sorted_subs_of_img, return_word_box=return_word_box)):
                    sub_img_id = sorted_subs_info[j]["sub_img_id"]
                    sub_img_info_list[sub_img_id]["rec_res"] = rec_res

                for sno in range(len(sub_img_info_list)):
                    rec_res = sub_img_info_list[sno]["rec_res"]
                    if rec_res["rec_score"] >= text_rec_score_thresh:
                        # Get per-char scores from patched predictor
                        char_scores = rec_res.get("rec_char_scores", [])
                        if return_word_box:
                            word_box_content_list, word_box_list = cal_ocr_word_box(
                                rec_res["rec_text"][0], dt_polys[sno], rec_res["rec_text"][1],
                            )
                            res["text_word"].append(word_box_content_list)
                            res["text_word_region"].append(word_box_list)
                            res["rec_texts"].append(rec_res["rec_text"][0])
                        else:
                            res["rec_texts"].append(rec_res["rec_text"])
                        res["rec_scores"].append(rec_res["rec_score"])
                        res["rec_char_scores"].append(char_scores if char_scores else [])
                        res["vis_fonts"].append(rec_res["vis_font"])
                        res["rec_polys"].append(dt_polys[sno])

        for res in results:
            if self.text_type == "general":
                rec_boxes = convert_points_to_boxes(res["rec_polys"])
                res["rec_boxes"] = rec_boxes
                if return_word_box:
                    res["text_word_boxes"] = [convert_points_to_boxes(line) for line in res["text_word_region"]]
            else:
                res["rec_boxes"] = np.array([])
            yield OCRResult(res)


_OCRPipeline.predict = _patched_pipeline_predict

app = FastAPI(title="OCR Service", description="Multi-engine OCR wrapper for Chinese character extraction")

# PaddleOCR engines, lazily loaded per language
_paddle_engines: dict[str, PaddleOCR] = {}

CJK_PATTERN = re.compile(r"[\u4e00-\u9fff]+")
LATIN_PATTERN = re.compile(r"[a-zA-Z]+")
VIET_PATTERN = re.compile(r"[a-zA-ZàáảãạăắằẳẵặâấầẩẫậèéẻẽẹêếềểễệìíỉĩịòóỏõọôốồổỗộơớờởỡợùúủũụưứừửữựỳýỷỹỵđÀÁẢÃẠĂẮẰẲẴẶÂẤẦẨẪẬÈÉẺẼẸÊẾỀỂỄỆÌÍỈĨỊÒÓỎÕỌÔỐỒỔỖỘƠỚỜỞỠỢÙÚỦŨỤƯỨỪỬỮỰỲÝỶỸỴĐ]+")

LANG_FILTER = {
    "zh": lambda text: bool(CJK_PATTERN.search(text)),
    "en": lambda text: bool(LATIN_PATTERN.search(text)) and not CJK_PATTERN.search(text),
    "vi": lambda text: bool(VIET_PATTERN.search(text)) and not CJK_PATTERN.search(text),
}


def _filter_ocr_by_language(raw: dict, language: str) -> dict:
    """Filter OCR raw output to keep only text matching the target language."""
    lang_check = LANG_FILTER.get(language)
    if not lang_check:
        return raw

    # paddleocr -> raw.lines, tesseract -> raw.words, google -> raw.annotations
    if "lines" in raw:
        raw["lines"] = [line for line in raw["lines"] if lang_check(line["text"])]
    elif "words" in raw:
        raw["words"] = [word for word in raw["words"] if lang_check(word["text"])]
    elif "annotations" in raw:
        # Keep first annotation (full text) + filtered rest
        if raw["annotations"]:
            raw["annotations"] = [raw["annotations"][0]] + [
                ann for ann in raw["annotations"][1:] if lang_check(ann["text"])
            ]
    return raw

# API language code → PaddleOCR language code
LANG_MAP = {"zh": "ch", "en": "en", "vi": "vi"}


class EngineEnum(str, Enum):
    paddleocr = "paddleocr"
    tesseract = "tesseract"
    google = "google"
    glm = "glm"
    deepseek = "deepseek"


class LanguageEnum(str, Enum):
    zh = "zh"
    en = "en"
    vi = "vi"


class ExtractRequest(BaseModel):
    image: str  # base64-encoded image
    language: LanguageEnum = LanguageEnum.zh
    engine: EngineEnum = EngineEnum.paddleocr


def _get_paddle_engine(language: str) -> PaddleOCR:
    paddle_lang = LANG_MAP.get(language, language)
    if paddle_lang not in _paddle_engines:
        _paddle_engines[paddle_lang] = PaddleOCR(lang=paddle_lang, use_textline_orientation=True)
    return _paddle_engines[paddle_lang]


def _segment_chinese(text: str) -> list[str]:
    """Segment Chinese text into words using jieba, keep only CJK words."""
    words = jieba.cut(text)
    result = []
    for word in words:
        cjk_only = "".join(CJK_PATTERN.findall(word))
        if cjk_only:
            result.append(cjk_only)
    return result


def _to_pinyin(text: str) -> str:
    """Convert Chinese text to space-separated pinyin with tone marks."""
    return " ".join(syllable[0] for syllable in pinyin(text, style=Style.TONE))


def _poly_to_location(poly) -> list[list[float]]:
    """Convert a polygon (numpy array or list of points) to [[x1,y1], [x2,y2], [x3,y3], [x4,y4]]."""
    if poly is None:
        return []
    try:
        pts = np.array(poly)
        if pts.ndim == 2 and pts.shape[0] >= 4 and pts.shape[1] >= 2:
            return [[round(float(pts[i][0])), round(float(pts[i][1]))] for i in range(4)]
    except (ValueError, TypeError):
        pass
    return []


def _extract_paddleocr(image_path: str, language: str) -> dict:
    engine = _get_paddle_engine(language)
    results = engine.predict(image_path)

    lines = []
    for result in results:
        texts = result.get("rec_texts", [])
        scores = result.get("rec_scores", [])
        char_scores_list = result.get("rec_char_scores", [[] for _ in texts])

        for text, score, char_scores in zip(texts, scores, char_scores_list):
            lines.append({
                "text": text,
                "score": round(float(score), 4),
                "char_scores": [round(float(s), 4) for s in char_scores] if char_scores else [],
            })

    return {"lines": lines}


def _extract_tesseract(image_path: str, language: str) -> dict:
    try:
        import pytesseract
    except ImportError:
        raise HTTPException(
            status_code=400,
            detail="Tesseract engine not available. Install: pip install pytesseract && brew install tesseract tesseract-lang",
        )

    tess_lang_map = {"zh": "chi_sim", "en": "eng", "vi": "vie"}
    tess_lang = tess_lang_map.get(language, "eng")

    from PIL import Image

    img = Image.open(image_path)
    data = pytesseract.image_to_data(img, lang=tess_lang, output_type=pytesseract.Output.DICT)

    words = []
    for i, text in enumerate(data["text"]):
        if data["level"][i] != 5:
            continue
        text = text.strip()
        if not text:
            continue
        words.append({
            "text": text,
            "conf": data["conf"][i],
        })

    return {"words": words}


ENGINES = {
    "paddleocr": _extract_paddleocr,
    "tesseract": _extract_tesseract,
    # "google" added below after _extract_google_vision is defined
}


def _extract_text_paddleocr(image_path: str, language: str) -> list[dict]:
    """Return raw text lines with bounding boxes — no word segmentation."""
    engine = _get_paddle_engine(language)
    results = engine.predict(image_path)

    blocks = []
    for result in results:
        texts = result.get("rec_texts", [])
        scores = result.get("rec_scores", [])
        rec_polys = result.get("rec_polys", [None] * len(texts))

        for text, score, poly in zip(texts, scores, rec_polys):
            text = text.strip()
            if not text:
                continue
            blocks.append({
                "text": text,
                "confidence": round(float(score), 4),
            })
    return blocks


def _extract_text_tesseract(image_path: str, language: str) -> list[dict]:
    """Return text lines with bounding boxes from Tesseract."""
    try:
        import pytesseract
    except ImportError:
        raise HTTPException(
            status_code=400,
            detail="Tesseract engine not available.",
        )

    tess_lang_map = {"zh": "chi_sim", "en": "eng", "vi": "vie"}
    tess_lang = tess_lang_map.get(language, "eng")

    from PIL import Image
    img = Image.open(image_path)
    data = pytesseract.image_to_data(img, lang=tess_lang, output_type=pytesseract.Output.DICT)

    # Group words by line
    lines: dict[tuple[int, int, int], list[dict]] = {}
    for i, text in enumerate(data["text"]):
        if data["level"][i] != 5:
            continue
        text = text.strip()
        if not text:
            continue
        key = (data["block_num"][i], data["par_num"][i], data["line_num"][i])
        if key not in lines:
            lines[key] = []
        lines[key].append({
            "text": text,
            "left": data["left"][i],
            "top": data["top"][i],
            "width": data["width"][i],
            "height": data["height"][i],
            "conf": data["conf"][i],
        })

    blocks = []
    for words in lines.values():
        line_text = " ".join(w["text"] for w in words)
        left = min(w["left"] for w in words)
        top = min(w["top"] for w in words)
        right = max(w["left"] + w["width"] for w in words)
        bottom = max(w["top"] + w["height"] for w in words)
        avg_conf = sum(max(0.0, w["conf"]) for w in words) / len(words) / 100.0
        blocks.append({
            "text": line_text,
            "confidence": round(avg_conf, 4),
        })
    return blocks


EXTRACT_TEXT_ENGINES = {
    "paddleocr": _extract_text_paddleocr,
    "tesseract": _extract_text_tesseract,
}


@app.post("/extract-text")
async def extract_text(req: ExtractRequest):
    if req.engine not in EXTRACT_TEXT_ENGINES:
        raise HTTPException(status_code=400, detail=f"Unknown engine: {req.engine}. Available: {list(EXTRACT_TEXT_ENGINES.keys())}")

    try:
        image_bytes = base64.b64decode(req.image)
    except Exception:
        raise HTTPException(status_code=400, detail="Invalid base64 image data")

    with tempfile.NamedTemporaryFile(suffix=".png", delete=True) as tmp:
        tmp.write(image_bytes)
        tmp.flush()

        blocks = EXTRACT_TEXT_ENGINES[req.engine](tmp.name, req.language)

    return {"blocks": blocks, "engine": req.engine}


@app.post("/recognize")
async def extract(req: ExtractRequest):
    if req.engine not in ENGINES:
        raise HTTPException(status_code=400, detail=f"Unknown engine: {req.engine}. Available: {list(ENGINES.keys())}")

    try:
        image_bytes = base64.b64decode(req.image)
    except Exception:
        raise HTTPException(status_code=400, detail="Invalid base64 image data")

    with tempfile.NamedTemporaryFile(suffix=".png", delete=True) as tmp:
        tmp.write(image_bytes)
        tmp.flush()

        characters = ENGINES[req.engine](tmp.name, req.language)

    return {"characters": characters, "engine": req.engine}


@app.get("/health")
async def health():
    return {"status": "ok", "engines": list(ENGINES.keys())}


# ---------------------------------------------------------------------------
# VLM engines (generative — no confidence/bounding box)
# ---------------------------------------------------------------------------

VLM_PROMPTS = {
    "zh": "请识别图片中的所有中文文本，忽略英文和其他语言。逐行列出，只输出中文，不要任何解释。",
    "en": "Recognize all English text in the image. Ignore Chinese and other languages. List line by line, output only English text, no explanation.",
    "vi": "Nhận dạng tất cả văn bản tiếng Việt trong ảnh. Bỏ qua tiếng Trung và tiếng Anh. Liệt kê từng dòng, chỉ xuất tiếng Việt, không giải thích.",
}


def _vlm_deepseek(image_path: str, language: str) -> dict:
    """Call DeepSeek VL API."""
    api_key = os.environ.get("DEEPSEEK_API_KEY", "")
    if not api_key:
        raise HTTPException(status_code=400, detail="Set DEEPSEEK_API_KEY env var. Get key at https://platform.deepseek.com")

    image_b64, mime = _encode_image_with_mime(image_path)
    prompt = VLM_PROMPTS.get(language, VLM_PROMPTS["zh"])
    payload = json.dumps({
        "model": "deepseek-chat",
        "messages": [{"role": "user", "content": [
            {"type": "image_url", "image_url": {"url": f"data:{mime};base64,{image_b64}"}},
            {"type": "text", "text": prompt},
        ]}],
        "max_tokens": 2048,
    }).encode()

    req = Request("https://api.deepseek.com/chat/completions", data=payload, method="POST", headers={
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
    })
    with urlopen(req, timeout=60) as resp:
        return json.loads(resp.read().decode())


def _vlm_glm(image_path: str, language: str) -> dict:
    """Call GLM-OCR via Ollama (local) or Z.AI cloud."""
    zai_key = os.environ.get("ZAI_API_KEY", "")

    prompt = VLM_PROMPTS.get(language, VLM_PROMPTS["zh"])

    if zai_key:
        # Cloud mode
        image_b64, mime = _encode_image_with_mime(image_path)
        payload = json.dumps({
            "model": "glm-4v-flash",
            "messages": [{"role": "user", "content": [
                {"type": "image_url", "image_url": {"url": f"data:{mime};base64,{image_b64}"}},
                {"type": "text", "text": prompt},
            ]}],
            "max_tokens": 2048,
        }).encode()

        req = Request("https://open.bigmodel.cn/api/paas/v4/chat/completions", data=payload, method="POST", headers={
            "Authorization": f"Bearer {zai_key}",
            "Content-Type": "application/json",
        })
        with urlopen(req, timeout=60) as resp:
            return json.loads(resp.read().decode())
    else:
        # Local Ollama mode
        image_b64 = _encode_image_b64(image_path)
        payload = json.dumps({
            "model": "glm-ocr",
            "messages": [{"role": "user", "content": prompt, "images": [image_b64]}],
            "stream": False,
        }).encode()

        req = Request("http://localhost:11434/api/chat", data=payload, method="POST", headers={
            "Content-Type": "application/json",
        })
        try:
            with urlopen(req, timeout=120) as resp:
                return json.loads(resp.read().decode())
        except URLError:
            raise HTTPException(
                status_code=400,
                detail="Cannot connect to Ollama. Run: ollama serve && ollama pull glm-ocr. Or set ZAI_API_KEY for cloud mode.",
            )


def _get_google_vision_client():
    """Get Google Cloud Vision client using service account from .gcp/ocr.json."""
    try:
        from google.cloud import vision
        from google.oauth2 import service_account
    except ImportError:
        raise HTTPException(
            status_code=400,
            detail="Install google-cloud-vision: pip install google-auth google-cloud-vision",
        )

    sa_path = os.path.join(os.path.dirname(__file__), "..", "..", ".gcp", "ocr.json")
    sa_path = os.path.abspath(sa_path)
    if not os.path.exists(sa_path):
        raise HTTPException(status_code=400, detail=f"Service account not found: {sa_path}")

    credentials = service_account.Credentials.from_service_account_file(
        sa_path, scopes=["https://www.googleapis.com/auth/cloud-platform"],
    )
    return vision.ImageAnnotatorClient(credentials=credentials)


def _extract_google_vision(image_path: str, language: str) -> dict:
    """Google Cloud Vision OCR — raw response."""
    from google.cloud import vision

    client = _get_google_vision_client()

    with open(image_path, "rb") as f:
        content = f.read()

    image = vision.Image(content=content)

    lang_hints = {"zh": ["zh"], "en": ["en"], "vi": ["vi"]}
    hints = lang_hints.get(language, [])
    image_context = vision.ImageContext(language_hints=hints) if hints else None

    response = client.text_detection(image=image, image_context=image_context)

    if response.error.message:
        raise HTTPException(status_code=500, detail=f"Google Vision API error: {response.error.message}")

    annotations = []
    for ann in response.text_annotations:
        verts = ann.bounding_poly.vertices
        annotations.append({
            "text": ann.description,
            "score": ann.score,
            "locale": ann.locale,
            "bounding_box": [[v.x, v.y] for v in verts] if verts else [],
        })

    return {"annotations": annotations}

    return characters


ENGINES["google"] = _extract_google_vision


def _encode_image_b64(image_path: str) -> str:
    with open(image_path, "rb") as f:
        return base64.b64encode(f.read()).decode()


def _encode_image_with_mime(image_path: str) -> tuple[str, str]:
    ext = image_path.rsplit(".", 1)[-1].lower()
    mime_map = {"png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg", "webp": "image/webp"}
    return _encode_image_b64(image_path), mime_map.get(ext, "image/png")


VLM_ENGINES = {
    "deepseek": _vlm_deepseek,
    "glm": _vlm_glm,
}

ALL_ENGINES = list(ENGINES.keys()) + list(VLM_ENGINES.keys())


# ---------------------------------------------------------------------------
# POST /test — file upload, all engines
# ---------------------------------------------------------------------------

@app.post("/test")
async def test_ocr(
    image: UploadFile = File(...),
    engine: EngineEnum = Form(EngineEnum.paddleocr),
    language: LanguageEnum = Form(LanguageEnum.zh),
):
    """
    Test OCR with file upload. Supports all engines:
      - paddleocr, tesseract (traditional OCR — returns characters with confidence)
      - deepseek, glm, google (VLM — returns raw text)

    curl examples:
      curl -F image=@photo.png -F engine=paddleocr http://localhost:8000/test
      curl -F image=@photo.png -F engine=glm http://localhost:8000/test
      curl -F image=@photo.png -F engine=google http://localhost:8000/test
      curl -F image=@photo.png -F engine=deepseek http://localhost:8000/test
    """
    if engine not in ENGINES and engine not in VLM_ENGINES:
        raise HTTPException(status_code=400, detail=f"Unknown engine: {engine}. Available: {ALL_ENGINES}")

    image_bytes = await image.read()
    if not image_bytes:
        raise HTTPException(status_code=400, detail="Empty image file")

    suffix = os.path.splitext(image.filename or "image.png")[1] or ".png"

    all_engines = {**ENGINES, **VLM_ENGINES}

    with tempfile.NamedTemporaryFile(suffix=suffix, delete=True) as tmp:
        tmp.write(image_bytes)
        tmp.flush()

        start = time.time()
        raw = all_engines[engine](tmp.name, language)
        elapsed = time.time() - start

    # Post-filter for OCR engines
    if engine in ENGINES:
        raw = _filter_ocr_by_language(raw, language)

    return {
        "engine": engine,
        "elapsed_seconds": round(elapsed, 2),
        "raw": raw,
    }
