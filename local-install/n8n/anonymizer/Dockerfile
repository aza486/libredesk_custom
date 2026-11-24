FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt \
    && python -m pip check \
    && python -c "import spacy; model = spacy.load('de_core_news_lg'); assert model.meta['version'] == '3.8.0'"

COPY analyze.py .

EXPOSE 8001

CMD ["uvicorn", "analyze:app", "--host", "0.0.0.0", "--port", "8001"]
