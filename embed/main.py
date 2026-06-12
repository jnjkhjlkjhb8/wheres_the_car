from fastapi import FastAPI
from sentence_transformers import SentenceTransformer

app = FastAPI()
model = SentenceTransformer("BAAI/bge-large-zh-v1.5")

@app.get("/health")
def health():
    return {"ok": True}

@app.post("/embed")
def embed(body: dict):
    texts = body.get("inputs", [])
    if not texts:
        return []
    vecs = model.encode(texts, normalize_embeddings=True)
    return vecs.tolist()
