
DOMAIN ?= api.gto.rakaoran.dev
DATA_PATH ?= ./certbot
EMAIL ?= 
STAGING ?= 0

.PHONY: help dummy-cert reload-nginx run-certbot

help:
	@echo "Available commands:"
	@echo "  make dummy-cert   - Create dummy certificates for initial setup"
	@echo "  make reload-nginx - Reload Nginx configuration (zero downtime)"
	@echo "  make run-certbot  - Run the certbot renewal loop (Requires EMAIL)"
	@echo "                      Usage: make run-certbot EMAIL=your@email.com [STAGING=1]"

dummy-cert:
	@echo "### Creating dummy certificate for $(DOMAIN) ..."
	@mkdir -p "$(DATA_PATH)/conf/live/$(DOMAIN)"
	@mkdir -p "$(DATA_PATH)/www"
	@openssl req -x509 -nodes -newkey rsa:4096 -days 1 \
		-keyout "$(DATA_PATH)/conf/live/$(DOMAIN)/privkey.pem" \
		-out "$(DATA_PATH)/conf/live/$(DOMAIN)/fullchain.pem" \
		-subj "/CN=localhost"
	@if [ ! -e "$(DATA_PATH)/conf/options-ssl-nginx.conf" ]; then \
		curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot-nginx/certbot_nginx/_internal/tls_configs/options-ssl-nginx.conf > "$(DATA_PATH)/conf/options-ssl-nginx.conf"; \
		curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot/certbot/ssl-dhparams.pem > "$(DATA_PATH)/conf/ssl-dhparams.pem"; \
	fi
	@echo "### Dummy certificate created at $(DATA_PATH)/conf/live/$(DOMAIN)"

reload-nginx:
	@echo "Reloading Nginx..."
	@docker exec $$(docker ps -q -f name=gto_nginx) nginx -s reload

run-certbot:
	@if [ -z "$(EMAIL)" ]; then echo "Usage: make run-certbot EMAIL=<EMAIL>"; exit 1; fi
	@echo "Starting Certbot loop for $(DOMAIN) with email $(EMAIL)..."
	@if [ "$(STAGING)" != "0" ]; then \
		echo "!!! RUNNING IN STAGING MODE (Fake Certs) !!!"; \
		STAGING_ARG="--staging"; \
	fi; \
	docker rm -f certbot-looper >/dev/null 2>&1 || true; \
	docker run -d \
	  --name certbot-looper \
	  --restart unless-stopped \
	  -v $$(pwd)/certbot/conf:/etc/letsencrypt \
	  -v $$(pwd)/certbot/www:/var/www/certbot \
	  --entrypoint "/bin/sh" \
	  certbot/certbot \
	  -c " \
		trap exit TERM; \
		if ! openssl x509 -in /etc/letsencrypt/live/$(DOMAIN)/fullchain.pem -noout -issuer | grep -q \"Let's Encrypt\"; then \
		  echo 'Found Dummy or Invalid Cert. Requesting real one...'; \
		  rm -rf /etc/letsencrypt/live/$(DOMAIN); \
		  rm -rf /etc/letsencrypt/archive/$(DOMAIN); \
		  rm -f /etc/letsencrypt/renewal/$(DOMAIN).conf; \
		  certbot certonly --webroot -w /var/www/certbot \
			-d $(DOMAIN) \
			--email $(EMAIL) \
			--agree-tos \
			--non-interactive \
			--force-renewal \
			$$STAGING_ARG; \
		else \
			echo \"Valid Let's Encrypt certificate found. Skipping initial request.\"; \
		fi; \
		while :; do \
		  echo 'Sleeping 12h...'; \
		  sleep 12h; \
		  echo 'Checking for renewal...'; \
		  certbot renew --webroot -w /var/www/certbot; \
		done \
	"
