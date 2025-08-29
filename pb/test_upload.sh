#!/bin/bash

# Test script to verify large file upload capability
# This creates a test audio file and uploads it to the PocketBase server

echo "🎵 Testing audio upload endpoint with large file..."

# Create a test file (100MB)
echo "Creating 100MB test file..."
dd if=/dev/zero of=test_audio.bin bs=1M count=100 2>/dev/null

# Get file size in human readable format
FILE_SIZE=$(ls -lh test_audio.bin | awk '{print $5}')
echo "Created test file: test_audio.bin ($FILE_SIZE)"

# Test the upload endpoint
echo "Testing upload to /api/ai/process-audio..."
curl -X POST http://localhost:8090/api/ai/process-audio \
  -H "Authorization: Bearer ra-dev-12345678901234567890123456789012" \
  -H "Content-Type: multipart/form-data" \
  -F "audio=@test_audio.bin;filename=test.wav" \
  -w "\n\nHTTP Status: %{http_code}\nTime: %{time_total}s\nUpload Speed: %{speed_upload} bytes/s\n" \
  -o response.json

echo "\nResponse:"
if [ -f response.json ]; then
  cat response.json | head -c 500
  echo "..."
fi

# Clean up
rm -f test_audio.bin response.json

echo "\n✅ Test complete!"