#!/bin/bash

USER_NAME=$(uuidgen | tr '-' 'a' | cut -c -8)
USER_PASS=$(openssl rand -base64 12)

echo "Generated Username: $USER_NAME"
echo "Generated Password: $USER_PASS"

# Optional: remove if you don't want to write to a file
echo $USER_NAME > credentials.txt
echo $USER_PASS >> credentials.txt