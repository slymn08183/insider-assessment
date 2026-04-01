#!/bin/bash
# Load test for event ingestion API
# Usage: bash test/load_test.sh
# Prereq: server running on :8080, hey installed (go install github.com/rakyll/hey@latest)

BASE_URL="http://localhost:8080"

echo "=== Health Check ==="
curl -s "$BASE_URL/health"
echo ""
echo ""

echo "=== Load Test: POST /events ==="
echo "Target: 20,000 requests, 50 concurrent workers"
echo ""

hey -n 20000 -c 200 -m POST \
  -H "Content-Type: application/json" \
  -D test/event_payload.json \
  "$BASE_URL/events"

echo ""
echo "=== Load Test: GET /metrics ==="
echo "Target: 1,000 requests, 50 concurrent workers"
echo ""

hey -n 1000 -c 50 \
  "$BASE_URL/metrics?event_name=product_view&from=2024-01-01T00:00:00Z&to=2025-12-31T00:00:00Z&group_by=daily"