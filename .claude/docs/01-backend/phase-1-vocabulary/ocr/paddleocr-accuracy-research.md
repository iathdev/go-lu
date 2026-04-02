# PaddleOCR Accuracy Improvement & Customization Research

**Date**: 2026-04-02
**Focus**: PP-OCRv5 configuration tuning, fine-tuning, preprocessing, custom training, limitations, and comparison with cloud APIs.

---

## 1. PP-OCRv5 Configuration Parameters That Affect Accuracy

### Core Configuration

```python
from paddleocr import PaddleOCR

ocr = PaddleOCR(
    use_doc_orientation_classify=True,   # Detect document rotation (0/90/180/270)
    use_doc_unwarping=True,              # Correct geometric distortion (UVDoc model)
    use_textline_orientation=True,       # Classify text line orientation
)
```

### Detection Parameters

| Parameter | Default | Notes |
|-----------|---------|-------|
| `text_det_limit_type` | `"min"` | How to constrain input image size. Options: `min`, `max` |
| `text_det_limit_side_len` | `736` | Side length limit. PP-OCRv5 changed default from 736 to 64 for better default performance. Higher values (960, 1280) improve accuracy for small text but increase latency |

**Speed vs accuracy tradeoffs for detection limit:**
- `max` + 640: Fastest (0.45s GPU on mobile)
- `max` + 960: Balanced
- `max` + 1280: Best accuracy for small text, slowest

### Recognition Parameters

| Parameter | Notes |
|-----------|-------|
| `rec_batch_num` | Number of text lines to batch for recognition. Higher = faster throughput, but too high may OOM |
| `max_text_length` | Upper limit for recognized text length. Training with increased values can cause models to plateau at ~50% accuracy |
| `character_dict_path` | Path to character dictionary. Must match between training and inference exactly |

### Auxiliary Modules Impact

Enabling auxiliary features (unwarping, orientation) increases resource usage:
- **Without** preprocessing: 0.62s GPU per image, 1054 chars/sec
- **With all** preprocessing: 1.09s GPU per image, 563 chars/sec
- More features do NOT always yield better results; evaluate per use case

### PP-OCRv5 Model Variants

| Variant | Best For |
|---------|----------|
| **Server** | GPU environments, higher accuracy |
| **Mobile** | CPU/edge devices, lower latency |

PP-OCRv5 changed defaults: both detection and recognition now use **server** models by default (previously mobile).

---

## 2. PP-OCRv5 Architecture & Accuracy Benchmarks

### Architecture

- **Detection backbone**: PP-HGNetV2 (replaced PP-HGNet from v4)
- **Recognition**: Dual-branch architecture
  - **GTC-NRTR branch**: Attention-based training (used during training only)
  - **SVTR-HGNet branch**: CTC loss, lightweight (used during inference)
- **Knowledge distillation**: GOT-OCR2.0 visual encoder as teacher model, PP-HGNetV2 as student
- **Model size**: 0.07B parameters, under 100MB total
- **Throughput**: 370+ chars/sec on Intel Xeon Gold 6271C CPU

### Accuracy Improvements (v4 -> v5)

| Scenario | PP-OCRv4 | PP-OCRv5 | Improvement |
|----------|----------|----------|-------------|
| Handwritten Chinese (server) | 0.3626 | 0.5807 | +60% relative |
| Printed English (server) | 0.6677 | 0.8679 | +30% relative |
| Japanese (server) | 0.4623 | 0.7372 | +59% relative |
| Server recognition average | 0.5735 | 0.8401 | +46% relative |
| Server detection average | 0.662 | 0.827 | +25% relative |
| Overall end-to-end | - | +13pp over v4 | - |

### OmniDocBench Results

PP-OCRv5 **ranks first** in average 1-edit distance across all scenarios, surpassing:
- GOT-OCR2.0-0.5B
- RolmOCR-7B
- Qwen2.5-VL-72B
- InternVL3-78B
- Gemini 2.5 Pro
- GPT-4o

Key advantage: No hallucination (unlike VLMs), precise bounding boxes.

---

## 3. Image Preprocessing Techniques

### Built-in PaddleOCR Preprocessing

1. **Document Orientation Classification** (PP-LCNet_x1_0_doc_ori)
   - Accuracy: 99.06% Top-1
   - Detects 0/90/180/270 degree rotation
   - GPU inference: ~2.62ms

2. **Text Image Unwarping** (UVDoc model)
   - Corrects geometric distortion from photography/scanning
   - Character Error Rate: 0.179
   - GPU inference: ~19.05ms

### External Preprocessing Pipeline (recommended before PaddleOCR)

Based on research papers testing preprocessing impact on PaddleOCR:

| Technique | Impact | When to Use |
|-----------|--------|-------------|
| **Grayscale conversion** | Moderate improvement | Always for color images |
| **Binarization (Otsu)** | Significant for scanned docs | Scanned documents with noisy backgrounds |
| **Adaptive thresholding** | Better than Otsu for uneven lighting | Photos of documents |
| **CLAHE** (Contrast Limited Adaptive Histogram Equalization) | Good for low-contrast text | Faded or low-contrast images |
| **Bilateral filtering** | Noise reduction while preserving edges | Noisy images |
| **Deskewing** | Critical for rotated text | Any non-horizontal text |
| **Erosion/Dilation** | Helps connect broken characters | Degraded/faded text |

### Recommended Preprocessing Pipeline

```
Input Image
  -> Grayscale conversion
  -> Noise reduction (bilateral filter or Gaussian blur)
  -> Contrast enhancement (CLAHE)
  -> Binarization (adaptive thresholding for photos, Otsu for scans)
  -> Deskew correction
  -> PaddleOCR inference
```

**Key insight**: While PaddleOCR works well on clean images, accuracy drops significantly with poor lighting, low resolution, or skewed angles. Preprocessing is critical for real-world robustness.

---

## 4. Fine-Tuning PaddleOCR

### When to Fine-Tune

- Domain-specific text (medical, legal, technical jargon)
- Non-standard fonts or handwriting styles
- Specific document layouts
- Languages/characters not well-covered by default model

### Dataset Requirements

| Component | Minimum Recommended |
|-----------|-------------------|
| Detection | 500+ labeled images |
| Recognition | Varies; 50 images can work for narrow domains |
| Data ratio (original:new) | 10:1 to 5:1 to avoid overfitting |

### Dataset Format

**Detection**: `image_path\t[{"transcription": "text", "points": [[x1,y1],[x2,y2],[x3,y3],[x4,y4]]}]`
- 8-point bounding box coordinates required
- Use PPOCRLabel for semi-automated annotation

**Recognition**: `image_path\ttext_label`
- Cropped text line images with corresponding text
- Character dictionary file listing all valid characters

### Key Hyperparameters

| Parameter | Fine-tuning Value | Notes |
|-----------|------------------|-------|
| `learning_rate` | 0.0001 (1/10 of training from scratch) | Lower LR prevents catastrophic forgetting |
| `epoch_num` | 5-10 (vs 500 from scratch) | Less epochs needed |
| `batch_size` | 128 (recognition) | Adjust based on GPU memory |
| `eval_batch_step` | [0, 500-2000] | Evaluate every N iterations |
| `save_epoch_step` | 50 | Checkpoint frequency |
| `pretrained_model` | PP-OCRv5 server weights | Download from PaddleOCR releases |

### Data Augmentation for Small Datasets

- **CopyPaste**: Paste text regions from other images
- **IaaAugment**: Random affine, noise, blur
- **RecAug**: Recognition-specific augmentation (can be removed if overfitting)
- Random rotation, blurring, geometric transformations

### Training Commands

```bash
# Fine-tune detection
python tools/train.py -c configs/det/your_det_config.yml

# Fine-tune recognition
python tools/train.py -c configs/rec/your_rec_config.yml

# Evaluate (uses Hmean/F1 as primary metric)
python tools/eval.py -c configs/det/your_det_config.yml

# Export for inference
python tools/export_model.py -c configs/rec/your_rec_config.yml \
    -o Global.pretrained_model=output/best_accuracy \
    Global.save_inference_dir=./inference_model
```

### Export produces three files:
- `inference.json` (model structure)
- `inference.pdiparams` (model weights)
- `inference.yaml` (configuration)

### Fine-Tuning Pitfalls

1. **Character dictionary mismatch**: Dictionary must be IDENTICAL between training and inference. Most common cause of accuracy loss after export.
2. **max_text_length misconfiguration**: Training with increased values can cause the model to plateau at ~50% accuracy.
3. **Export accuracy loss**: Accuracy can drop significantly after exporting to inference model. Verify inference accuracy matches training accuracy.
4. **Overfitting on small datasets**: Use augmentation and maintain original:new data ratio of 10:1 to 5:1.
5. **Python version**: PaddlePaddle requires Python 3.7-3.9 (strict).

---

## 5. Custom Training from Scratch vs Fine-Tuning

| Aspect | Fine-Tuning | Training from Scratch |
|--------|------------|---------------------|
| **Data needed** | 500+ images (det), 50+ (rec) | 10,000+ images |
| **Compute time** | Hours (5-10 epochs) | Days-weeks (500+ epochs) |
| **GPU requirement** | Single GPU sufficient | Multi-GPU recommended |
| **Accuracy** | Good for domain adaptation | Better for completely novel scripts |
| **Risk** | Overfitting if data too small | Underfitting if data insufficient |
| **Recommendation** | **Start here for most use cases** | Only if fine-tuning insufficient |

### For CJK and Handwritten Chinese specifically:

PP-OCRv5 already includes:
- HWDB dataset (3.9M handwritten Chinese samples, 7356 categories)
- Synthetic handwritten samples using ERNIE-4.5-VL
- Hard case mining for rare characters
- Unified Simplified/Traditional Chinese + Japanese + Pinyin model

**Recommendation**: Fine-tune PP-OCRv5 server model rather than training from scratch. The v5 model already has strong CJK handwriting baseline (0.5807 accuracy for handwritten Chinese).

---

## 6. Known Limitations and Workarounds

### Accuracy Limitations

| Limitation | Workaround |
|-----------|-----------|
| Handwriting accuracy (58% on Chinese) still not production-grade for critical data entry | Combine with confidence thresholding + human review for low-confidence results |
| Complex layouts (tables, multi-column) | Use PaddleX layout analysis pipeline |
| Very small text | Increase `text_det_limit_side_len` to 960-1280 |
| Rotated text in mixed-orientation docs | Enable `use_textline_orientation=True` |
| Warped/curved document photos | Enable `use_doc_unwarping=True` |
| Rare/ancient Chinese characters | Fine-tune with domain-specific data |

### Production Deployment Issues

1. **Cold start overhead**: 4.2s startup time. Keep models warm in production.
2. **Memory usage**: Lightweight (~100MB model), but multiple auxiliary modules add up.
3. **Inference vs training accuracy gap**: Always validate with inference model, not training checkpoints.
4. **PaddlePaddle dependency**: Heavier than pure PyTorch; consider ONNX export for lighter deployment.
5. **Multi-instance**: Stateless inference, works fine with horizontal scaling.

### Configuration Gotchas

- `limit_side_len` changed from 736 to 64 in v5 defaults -- verify this matches your use case
- Server model is now default (previously mobile) -- be aware of increased compute
- Larger dictionary in v5 increases inference time slightly

---

## 7. PaddleOCR vs Cloud APIs

### Accuracy Comparison

| Solution | FUNSD Accuracy | STROIE Accuracy | Notes |
|----------|---------------|-----------------|-------|
| Google Cloud Vision AI | 75.0% | 87.8% | Best overall performer |
| Azure AI Document Intelligence | Competitive | Competitive | Best value at $0.50/1K pages |
| PaddleOCR | Mixed | Mixed | Best open-source trade-off |
| Tesseract | Lower | Lower | Lowest accuracy |

### Cost Comparison

| Solution | Cost per 1,000 pages | Notes |
|----------|---------------------|-------|
| Google Cloud Vision AI | $1.50 | Best accuracy |
| Amazon Textract | $1.50 | Good for AWS ecosystem |
| Azure Document Intelligence | $0.50 | Best commercial value |
| PaddleOCR (self-hosted, A100 GPU) | ~$0.09 | **16x cheaper** than cloud |
| PaddleOCR (CPU) | ~$0.02-0.05 | Cheapest option |

### OmniDocBench: PP-OCRv5 vs VLMs

On the OmniDocBench benchmark, PP-OCRv5 **outperforms all tested VLMs** (including Gemini 2.5 Pro, GPT-4o) in pure OCR accuracy. However, cloud APIs add value through:
- Layout understanding
- Table extraction
- Form field detection
- Handwriting specialization (Google)

### Decision Matrix

| Scenario | Recommendation |
|----------|---------------|
| **High volume, printed text, cost-sensitive** | PaddleOCR (self-hosted) |
| **Complex documents, tables, forms** | Google Cloud Vision or Azure |
| **Handwritten Chinese (critical accuracy)** | Google Cloud Vision + human review |
| **Handwritten Chinese (acceptable accuracy)** | PaddleOCR v5 fine-tuned + confidence thresholding |
| **Edge/mobile deployment** | PaddleOCR mobile model |
| **Mixed: printed + handwritten** | PaddleOCR for printed, cloud API for handwritten (hybrid) |
| **CJK printed text** | PaddleOCR v5 server (already strong) |
| **Multilingual (5+ languages)** | PaddleOCR v5 (unified model supports 40+ languages) |

### Is It Worth Customizing PaddleOCR vs Using Cloud APIs?

**Worth customizing PaddleOCR when:**
- Processing >10K pages/month (cost savings compound)
- Privacy/data sovereignty requirements
- Consistent document types (invoices, ID cards, receipts)
- Edge/offline deployment needed
- Printed CJK text (already very strong baseline)

**Use cloud APIs when:**
- Handwritten text is critical and accuracy must be >90%
- Complex/variable document layouts
- Low volume (<1K pages/month, marginal cost difference)
- Need structured extraction (tables, forms, key-value pairs)
- Don't want to maintain ML infrastructure

**Hybrid approach (recommended for our use case):**
- Use PaddleOCR for initial/bulk processing of printed text
- Route low-confidence results to Google Cloud Vision API
- Fine-tune PaddleOCR on domain-specific vocabulary flashcard images over time
- Gradually reduce cloud API dependency as PaddleOCR accuracy improves

---

## Sources

- [PP-OCRv5 Official Documentation](http://www.paddleocr.ai/main/en/version3.x/algorithm/PP-OCRv5/PP-OCRv5.html)
- [PaddleOCR 3.0 Technical Report (arXiv)](https://arxiv.org/html/2507.05595v1)
- [PP-OCRv5 on Hugging Face (Baidu Blog)](https://huggingface.co/blog/baidu/ppocrv5)
- [OCR Fine-Tuning: From Raw Data to Custom Paddle OCR Model (HackerNoon)](https://hackernoon.com/ocr-fine-tuning-from-raw-data-to-custom-paddle-ocr-model)
- [Fine-tuning PaddleOCR models for text recognition (tim's blog)](https://timc.me/blog/finetune-paddleocr-text-recognition.html)
- [Train Your Own OCR Model with PaddleOCR (DataGet)](https://dataget.ai/blogs/train-your-own-ocr-model-paddleocr/)
- [Impact of image pre-processing on PaddleOCR performance (ScienceDirect)](https://www.sciencedirect.com/science/article/pii/S1877050925027383)
- [PaddleX Document Image Preprocessing Pipeline](https://paddlepaddle.github.io/PaddleX/3.3/en/pipeline_usage/tutorials/ocr_pipelines/doc_preprocessor.html)
- [Identifying the Best OCR API: Benchmarking OCR APIs (Nanonets)](https://nanonets.com/blog/identifying-the-best-ocr-api/)
- [PaddleOCR vs Tesseract Benchmark (TildAlice)](https://tildalice.io/ocr-tesseract-easyocr-paddleocr-benchmark/)
- [What It Really Takes to Use PaddleOCR in Production (Medium)](https://medium.com/@ankitladva11/what-it-really-takes-to-use-paddleocr-in-production-systems-d63e38ded55e)
- [OCR Technologies Compared (Medium)](https://medium.com/@ilia.ozhmegov/ocr-technologies-compared-tesseract-abbyy-ocr-google-cloud-vision-paddle-ocr-easyocr-and-6312cf1c2ea7)
- [Comparing Top 6 OCR Systems 2025 (MarkTechPost)](https://www.marktechpost.com/2025/11/02/comparing-the-top-6-ocr-optical-character-recognition-models-systems-in-2025/)
- [Open-Source OCR Models 2025 Benchmarks (E2E Networks)](https://www.e2enetworks.com/blog/complete-guide-open-source-ocr-models-2025)
- [Fine-tune v3 detection model on handwriting (GitHub Discussion)](https://github.com/PaddlePaddle/PaddleOCR/discussions/14868)
- [Loss of accuracy when exporting model (GitHub Issue)](https://github.com/PaddlePaddle/PaddleOCR/issues/11551)
- [PaddleOCR GitHub Repository](https://github.com/PaddlePaddle/PaddleOCR)
