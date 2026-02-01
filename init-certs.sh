#!/bin/bash


domain="example.com"
data_path="./certbot"

echo "### Creating dummy certificate for $domain ..."

mkdir -p "$data_path/conf/live/$domain"
mkdir -p "$data_path/www"

# Generate the fake cert locally
openssl req -x509 -nodes -newkey rsa:4096 -days 1 \
  -keyout "$data_path/conf/live/$domain/privkey.pem" \
  -out "$data_path/conf/live/$domain/fullchain.pem" \
  -subj "/CN=localhost"

# Download TLS parameters
if [ ! -e "$data_path/conf/options-ssl-nginx.conf" ]; then
  curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot-nginx/certbot_nginx/_internal/tls_configs/options-ssl-nginx.conf > "$data_path/conf/options-ssl-nginx.conf"
  curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot/certbot/ssl-dhparams.pem > "$data_path/conf/ssl-dhparams.pem"
fi

echo "### Dummy certificate created at $data_path/conf/live/$domain"