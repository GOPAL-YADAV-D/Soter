#!/bin/bash

# Source the test script functions
source scripts/test-features.sh

# Get tokens
CSRF_TOKEN=$(curl -s -c /tmp/test_cookies.txt http://localhost:8080/csrf-token | grep -o '"csrf_token":"[^"]*"' | cut -d'"' -f4)
JWT_TOKEN=$(curl -s -b /tmp/test_cookies.txt -X POST -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF_TOKEN" -d '{"email":"test2@example.com","password":"testpassword123"}' http://localhost:8080/api/v1/auth/login | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

echo "CSRF_TOKEN: $CSRF_TOKEN"
echo "JWT_TOKEN: ${JWT_TOKEN:0:50}..."
echo "COOKIE_JAR: $COOKIE_JAR"

# Test the api_request function
echo "Testing api_request function..."
UPLOAD_DATA='{
    "files": [
        {
            "filename": "test-upload.txt",
            "mimeType": "text/plain",
            "fileSize": 50,
            "folderPath": "/",
            "contentHash": ""
        }
    ],
    "totalBytes": 50
}'

echo "Calling api_request..."
RESULT=$(api_request "POST" "/api/v1/files/upload-session" "$UPLOAD_DATA")
echo "Result: $RESULT"