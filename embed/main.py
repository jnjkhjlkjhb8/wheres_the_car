import logging
import time
from fastapi import FastAPI
from sentence_transformers import SentenceTransformer

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger(__name__)
log.info("loading model BAAI/bge-large-zh-v1.5")
model = SentenceTransformer("BAAI/bge-large-zh-v1.5")
log.info("model loaded")
app = FastAPI()

@app.get("/health")
def health():
    return {"ok": True}

@app.post("/embed")
def embed(body: dict):
    texts = body.get("inputs", [])
    if not texts:
        return []
    t0 = time.monotonic()
    vecs = model.encode(texts, normalize_embeddings=True)
    elapsed = (time.monotonic() - t0) * 1000
    log.info("embed count=%d elapsed=%.0fms", len(texts), elapsed)
    return vecs.tolist()
