#!/bin/bash

# 1. INPUT CHECK
if [ -z "$1" ]; then
  echo "Usage: ./2-run-certbot.sh <EMAIL>"
  exit 1
fi

# 2. CONFIGURATION
DOMAIN="api.gto.rakaoran.dev"
EMAIL=$1
STAGING=0

if [ "$STAGING" != "0" ]; then 
  STAGING_ARG="--staging"; 
  echo "!!! RUNNING IN STAGING MODE (Fake Certs) !!!"
fi

docker run -d \
  --name certbot-looper \
  --restart unless-stopped \
  -v $(pwd)/certbot/conf:/etc/letsencrypt \
  -v $(pwd)/certbot/www:/var/www/certbot \
  --entrypoint "/bin/sh" \
  certbot/certbot \
  -c "
    trap exit TERM;
    
    # Check if we need to replace the cert
    if ! openssl x509 -in /etc/letsencrypt/live/$DOMAIN/fullchain.pem -noout -issuer | grep -q \"Let's Encrypt\"; then
      echo 'Found Dummy or Invalid Cert. Requesting real one...'
      
      # Clean up dummy files cleanly
      rm -rf /etc/letsencrypt/live/$DOMAIN
      rm -rf /etc/letsencrypt/archive/$DOMAIN
      rm -f /etc/letsencrypt/renewal/$DOMAIN.conf
      
      certbot certonly --webroot -w /var/www/certbot \
        -d $DOMAIN \
        --email $EMAIL \
        --agree-tos \
        --non-interactive \
        --force-renewal \
        $STAGING_ARG
    else
        echo 'Valid Let\'s Encrypt certificate found. Skipping initial request.'
    fi

    # Renewal loop
    while :; do
      echo 'Sleeping 12h...'
      sleep 12h
      echo 'Checking for renewal...'
      certbot renew --webroot -w /var/www/certbot
    done
"