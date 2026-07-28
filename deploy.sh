#!/usr/bin/env bash
set -euo pipefail

REPO_URL="git@github.com:S0heilNG/voice-travel-assistant.git"
BASE_DIR="/var/www/vta"
RELEASES_DIR="$BASE_DIR/releases"
SHARED_ENV="$BASE_DIR/shared/.env"
CURRENT_LINK="$BASE_DIR/current"
BACKEND_SERVICE="vta-backend"
AUTOMATION_SERVICE="vta-automation"
KEEP_RELEASES=3
BACKEND_HEALTH_URL="http://localhost:8080/health"
AUTOMATION_HEALTH_URL="http://localhost:4000/health"
# Nginx serves the SPA from $CURRENT_LINK/frontend/dist, so this fetches the
# index.html of whichever release is currently linked.
FRONTEND_HEALTH_URL="http://localhost/"
GO_BIN="/usr/local/go/bin/go"

BRANCH="${1:-main}"
RELEASE_NAME="v-$(date +%Y%m%d-%H%M)"
RELEASE_PATH="$RELEASES_DIR/$RELEASE_NAME"

echo "==> Deploying release '$RELEASE_NAME' from branch '$BRANCH'"

if [ ! -f "$SHARED_ENV" ]; then
    echo "ERROR: shared env file not found at $SHARED_ENV" >&2
    exit 1
fi

if [ -e "$RELEASE_PATH" ]; then
    echo "ERROR: release path $RELEASE_PATH already exists" >&2
    exit 1
fi

# Capture the release currently live (if any) so we can roll back to it.
PREVIOUS_RELEASE=""
if [ -L "$CURRENT_LINK" ]; then
    PREVIOUS_RELEASE="$(readlink -f "$CURRENT_LINK")"
fi

echo "==> Cloning $REPO_URL (branch: $BRANCH)"
CLONE_ATTEMPTS=5
for attempt in $(seq 1 "$CLONE_ATTEMPTS"); do
    if git clone --branch "$BRANCH" --single-branch "$REPO_URL" "$RELEASE_PATH"; then
        break
    fi
    rm -rf "$RELEASE_PATH"
    if [ "$attempt" -eq "$CLONE_ATTEMPTS" ]; then
        echo "ERROR: git clone failed after $CLONE_ATTEMPTS attempts (flaky network to github.com)" >&2
        exit 1
    fi
    echo "==> Clone attempt $attempt failed, retrying in 5s..."
    sleep 5
done

echo "==> Building backend"
(
    cd "$RELEASE_PATH/backend"
    "$GO_BIN" build -o vta-backend ./cmd/server
)

echo "==> Installing shared .env"
cp "$SHARED_ENV" "$RELEASE_PATH/backend/.env"

# The frontend is built into the release, and Nginx's root points at
# $CURRENT_LINK/frontend/dist. That means the atomic symlink switch below
# swaps the backend and the frontend together, so the served SPA can never
# drift out of sync with the deployed backend (it used to: Nginx served a
# separate, hand-copied directory that no deploy ever touched).
#
# VITE_API_BASE is deliberately the empty string, not unset: api.js uses `??`
# so "" means "same origin" (Nginx proxies /api/), while unset would fall
# back to http://localhost:8080 and break every browser that isn't the server.
echo "==> Building frontend"
(
    cd "$RELEASE_PATH/frontend"
    npm ci --no-audit --no-fund
    VITE_API_BASE="" npm run build
)

if [ ! -f "$RELEASE_PATH/frontend/dist/index.html" ]; then
    echo "ERROR: frontend build produced no dist/index.html" >&2
    exit 1
fi

# Remember the hashed bundle name so the post-switch health check can prove
# Nginx is really serving *this* release's build, not a stale one.
BUILT_ASSET="$(basename "$(ls -1 "$RELEASE_PATH"/frontend/dist/assets/*.js | head -1)")"
echo "==> Frontend bundle: $BUILT_ASSET"

# node_modules is build scratch (~40M); dist is the only artifact Nginx needs.
rm -rf "$RELEASE_PATH/frontend/node_modules"

if [ -f "$RELEASE_PATH/automation-service/package.json" ]; then
    DEP_COUNT="$(node -e "const p=require('$RELEASE_PATH/automation-service/package.json'); process.stdout.write(String(Object.keys(p.dependencies||{}).length))")"
    if [ "$DEP_COUNT" -gt 0 ]; then
        echo "==> Installing automation-service dependencies ($DEP_COUNT found)"
        (
            cd "$RELEASE_PATH/automation-service"
            npm install --production
        )
    else
        echo "==> automation-service has no dependencies yet, skipping npm install"
    fi
fi

echo "==> Switching 'current' symlink to $RELEASE_NAME"
ln -sfn "$RELEASE_PATH" "$CURRENT_LINK"

echo "==> Restarting $BACKEND_SERVICE and $AUTOMATION_SERVICE"
sudo -n systemctl restart "$BACKEND_SERVICE"
sudo -n systemctl restart "$AUTOMATION_SERVICE"

echo "==> Waiting for services to come up"
sleep 2

HEALTH_OK=true

echo "==> Health check: $BACKEND_HEALTH_URL"
if curl -sf "$BACKEND_HEALTH_URL" > /dev/null; then
    echo "==> $BACKEND_SERVICE health check passed"
else
    echo "==> $BACKEND_SERVICE health check FAILED"
    HEALTH_OK=false
fi

echo "==> Health check: $AUTOMATION_HEALTH_URL"
if curl -sf "$AUTOMATION_HEALTH_URL" > /dev/null; then
    echo "==> $AUTOMATION_SERVICE health check passed"
else
    echo "==> $AUTOMATION_SERVICE health check FAILED"
    HEALTH_OK=false
fi

# Frontend check: assert the *served* index.html references this release's
# hashed bundle. A plain 200 would not catch the stale-frontend bug, since
# the old build also returned 200 quite happily.
echo "==> Health check: $FRONTEND_HEALTH_URL (expecting $BUILT_ASSET)"
if curl -sf "$FRONTEND_HEALTH_URL" | grep -q "$BUILT_ASSET"; then
    echo "==> frontend health check passed"
else
    echo "==> frontend health check FAILED — served index.html does not reference $BUILT_ASSET"
    HEALTH_OK=false
fi

if [ "$HEALTH_OK" != true ]; then
    echo "==> Health check FAILED — rolling back"
    if [ -n "$PREVIOUS_RELEASE" ] && [ -d "$PREVIOUS_RELEASE" ]; then
        ln -sfn "$PREVIOUS_RELEASE" "$CURRENT_LINK"
        sudo -n systemctl restart "$BACKEND_SERVICE"
        sudo -n systemctl restart "$AUTOMATION_SERVICE"
        echo "ERROR: deploy of $RELEASE_NAME failed health check; rolled back to $(basename "$PREVIOUS_RELEASE")" >&2
    else
        echo "ERROR: deploy of $RELEASE_NAME failed health check; no previous release available to roll back to" >&2
    fi
    exit 1
fi

echo "==> Pruning old releases (keeping last $KEEP_RELEASES)"
CURRENT_TARGET_NAME="$(basename "$(readlink -f "$CURRENT_LINK")")"
mapfile -t ALL_RELEASES < <(ls -1 "$RELEASES_DIR" | sort)
TOTAL=${#ALL_RELEASES[@]}
if [ "$TOTAL" -gt "$KEEP_RELEASES" ]; then
    TO_REMOVE=$((TOTAL - KEEP_RELEASES))
    for i in $(seq 0 $((TO_REMOVE - 1))); do
        OLD_RELEASE="${ALL_RELEASES[$i]}"
        if [ "$OLD_RELEASE" = "$CURRENT_TARGET_NAME" ]; then
            continue
        fi
        echo "    removing old release: $OLD_RELEASE"
        rm -rf "${RELEASES_DIR:?}/${OLD_RELEASE:?}"
    done
fi

echo "==> Deploy complete"
echo "Release: $RELEASE_NAME"
echo "Health check: OK"
