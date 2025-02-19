FROM python:3.12.8 AS base
WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY data/ ./data
COPY test_data_processing.py .
COPY data_processing.py .

FROM base AS test
RUN pip install pytest-watch
CMD ["ptw", "--ext=.py", "--runner=pytest"]

FROM base AS production
CMD ["python", "data_processing.py"]
