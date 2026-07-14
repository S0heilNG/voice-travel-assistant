#!/usr/bin/env bash
set -euo pipefail

REPO_URL="git@github.com:S0heilNG/voice-travel-assistant.git"
BASE_DIR="/var/www/vta"
RELEASES_DIR="$BASE_DIR/releases"
SHARED_ENV="$BASE_DIR/shared/.env"
CURRENT_LINK="$BASE_DIR/current"
SERVICE_NAME="vta-backend"
KEEP_RELEASES=3
HEALTH_URL="http://localhost:8080/health"
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

echo "==> Restarting $SERVICE_NAME"
sudo -n systemctl restart "$SERVICE_NAME"

echo "==> Waiting for service to come up"
sleep 2

echo "==> Health check: $HEALTH_URL"
if curl -sf "$HEALTH_URL" > /dev/null; then
    echo "==> Health check passed"
else
    echo "==> Health check FAILED — rolling back"
    if [ -n "$PREVIOUS_RELEASE" ] && [ -d "$PREVIOUS_RELEASE" ]; then
        ln -sfn "$PREVIOUS_RELEASE" "$CURRENT_LINK"
        sudo -n systemctl restart "$SERVICE_NAME"
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
