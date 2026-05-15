FROM python:3.12-slim

# Non-root user for security
RUN groupadd -r vault && useradd -r -g vault -m -d /app vault

WORKDIR /app

# Install dependencies as root before switching user
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy app files
COPY app.py logic.py reset.py ./
COPY templates/ ./templates/

# Create uploads dir owned by vault user
RUN mkdir -p uploads && chown -R vault:vault /app

# Switch to non-root
USER vault

EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:5000')" || exit 1

CMD ["python3", "app.py"]
