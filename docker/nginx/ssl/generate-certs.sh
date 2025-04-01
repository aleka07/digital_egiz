#!/bin/bash
#
# Script to generate self-signed SSL certificates for development
#

# Set variables
DOMAIN="localhost"
CERT_PATH="."
DAYS_VALID=365
KEY_SIZE=2048

# Generate self-signed certificate
echo "Generating self-signed SSL certificate for $DOMAIN..."
openssl req -x509 -nodes -newkey rsa:$KEY_SIZE -days $DAYS_VALID \
    -keyout "$CERT_PATH/nginx-selfsigned.key" \
    -out "$CERT_PATH/nginx-selfsigned.crt" \
    -subj "/CN=$DOMAIN/O=Digital Egiz/C=US" \
    -addext "subjectAltName = DNS:$DOMAIN,DNS:www.$DOMAIN,IP:127.0.0.1"

# Set permissions
chmod 644 "$CERT_PATH/nginx-selfsigned.crt"
chmod 600 "$CERT_PATH/nginx-selfsigned.key"

echo "SSL certificate generation complete."
echo "Certificate: $CERT_PATH/nginx-selfsigned.crt"
echo "Private key: $CERT_PATH/nginx-selfsigned.key"
echo
echo "IMPORTANT: This is a self-signed certificate for development use only."
echo "           In production, use certificates from a trusted certificate authority." 