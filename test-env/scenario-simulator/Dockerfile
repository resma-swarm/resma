FROM python:3.12-slim

WORKDIR /app

RUN pip install --no-cache-dir requests

COPY scenario-simulator.py .

CMD ["python", "-u", "scenario-simulator.py"]
