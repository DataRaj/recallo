#!/usr/bin/env bash
set -euo pipefail

# Run as root on a fresh DigitalOcean Ubuntu droplet.
# Usage: bash setup.sh <your-domain>
DOMAIN="${1:?Usage: setup.sh <domain>}"

# ── 1. Deploy user ─────────────────────────────────────────────────────────────
id deploy &>/dev/null || adduser --disabled-password --gecos "" deploy
usermod -aG sudo deploy
[ -d /home/deploy/.ssh ] || rsync --archive --chown=deploy:deploy ~/.ssh /home/deploy

# ── 2. Firewall ────────────────────────────────────────────────────────────────
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# ── 3. Packages ────────────────────────────────────────────────────────────────
apt-get update -q
apt-get install -y -q redis-server nginx certbot python3-certbot-nginx

# ── 4. Secure Redis ────────────────────────────────────────────────────────────
REDIS_CONF=/etc/redis/redis.conf
sed -i 's/^bind .*/bind 127.0.0.1/' "$REDIS_CONF"
if ! grep -q "^requirepass " "$REDIS_CONF"; then
    REDIS_PASS=$(openssl rand -hex 32)
    echo "requirepass $REDIS_PASS" >> "$REDIS_CONF"
    echo "[setup] Redis password: $REDIS_PASS  — add to /opt/airstage/.env as REDIS_URL"
fi
systemctl enable redis-server
systemctl restart redis-server

# ── 5. App directories ────────────────────────────────────────────────────────
mkdir -p /opt/airstage/bin
chown -R deploy:deploy /opt/airstage

# ── 6. Nginx ──────────────────────────────────────────────────────────────────
NGINX_CONF=/etc/nginx/sites-available/airstage
cp "$(dirname "$0")/../nginx/airstage" "$NGINX_CONF"
sed -i "s/api.yourdomain.com/$DOMAIN/g" "$NGINX_CONF"
ln -sf "$NGINX_CONF" /etc/nginx/sites-enabled/airstage
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl reload nginx

# ── 7. TLS ────────────────────────────────────────────────────────────────────
certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos -m "ops@$DOMAIN" --redirect

# ── 8. Systemd unit ───────────────────────────────────────────────────────────
cp "$(dirname "$0")/../systemd/airstage-api.service" /etc/systemd/system/
systemctl daemon-reload
systemctl enable airstage-api

echo ""
echo "[setup] Done. Next:"
echo "  1. SCP your .env to /opt/airstage/.env (owned deploy:deploy, chmod 600)"
echo "  2. SCP the binary to /opt/airstage/bin/airstage"
echo "  3. systemctl start airstage-api"
echo "  4. journalctl -u airstage-api -f"
