FROM python:3.12-slim

WORKDIR /app

RUN pip install --no-cache-dir requests

COPY load-generator.py .

CMD ["python", "-u", "load-generator.py"]
