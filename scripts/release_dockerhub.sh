#!/usr/bin/env bash
set -euo pipefail

# Build and publish azs3-proxy images to Docker Hub.
# Requirements: docker installed and running; Docker Hub credentials via env:
#   DOCKERHUB_USERNAME  (Docker Hub username or org)
#   DOCKERHUB_TOKEN     (Docker Hub access token or password)
# Optional overrides:
#   IMAGE_NAME          (default: azs3-proxy)
#   IMAGE_NAMESPACE     (default: $DOCKERHUB_USERNAME)
#   IMAGE_TAG           (default: version from internal/version/version.go)
#   PUSH_LATEST         (default: true; set to false to skip latest tag)
#   DOCKERHUB_PRIVATE   (default: true; set false to skip making repo private)
#   DOCKER_BUILD_ARGS   (default: --network=host; additional args for docker build)

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="$ROOT_DIR/internal/version/version.go"

# Load .env if present
if [[ -f "$ROOT_DIR/.env" ]]; then
  # shellcheck disable=SC1090
  source "$ROOT_DIR/.env"
fi

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required tool: $1" >&2
    exit 127
  fi
}

need docker
need curl
need python3

if [[ -z "${DOCKERHUB_USERNAME:-}" || -z "${DOCKERHUB_TOKEN:-}" ]]; then
  echo "Set DOCKERHUB_USERNAME and DOCKERHUB_TOKEN (Docker Hub access token)." >&2
  exit 1
fi

if [[ ! -f "$VERSION_FILE" ]]; then
  echo "Version file not found at $VERSION_FILE" >&2
  exit 1
fi

VERSION_RAW=$(grep 'Version = "' "$VERSION_FILE" | head -n1 | sed 's/.*Version = "\(.*\)"/\1/')
if [[ -z "$VERSION_RAW" ]]; then
  echo "Could not extract version from $VERSION_FILE" >&2
  exit 1
fi
VERSION="${IMAGE_TAG:-$VERSION_RAW}"
IMAGE_NAME="${IMAGE_NAME:-azs3-proxy}"
IMAGE_NAMESPACE="${IMAGE_NAMESPACE:-$DOCKERHUB_USERNAME}"
PUSH_LATEST="${PUSH_LATEST:-true}"
DOCKERHUB_PRIVATE="${DOCKERHUB_PRIVATE:-true}"
DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS:---network=host}"

IMAGE_BASE="docker.io/${IMAGE_NAMESPACE}/${IMAGE_NAME}"

echo "Logging in to Docker Hub as ${DOCKERHUB_USERNAME}..."
echo "$DOCKERHUB_TOKEN" | docker login --username "$DOCKERHUB_USERNAME" --password-stdin

if [[ "$DOCKERHUB_PRIVATE" == "true" ]]; then
  echo "Ensuring Docker Hub repo ${IMAGE_NAMESPACE}/${IMAGE_NAME} is private..."
  LOGIN_JSON=$(printf '{"username":"%s","password":"%s"}' "$DOCKERHUB_USERNAME" "$DOCKERHUB_TOKEN")
  JWT=$(echo "$LOGIN_JSON" | curl -s -X POST -H "Content-Type: application/json" -d @- https://hub.docker.com/v2/users/login | python3 -c 'import sys,json; print(json.load(sys.stdin).get("token",""))')
  if [[ -z "$JWT" ]]; then
    echo "Failed to obtain Docker Hub JWT; cannot set repo privacy." >&2
    exit 1
  fi
  CREATE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "Authorization: JWT $JWT" \
    -H "Content-Type: application/json" \
    -d "{\"namespace\":\"$IMAGE_NAMESPACE\",\"name\":\"$IMAGE_NAME\",\"is_private\":true}" \
    https://hub.docker.com/v2/repositories/)
  if [[ "$CREATE_CODE" == "201" ]]; then
    : # created successfully
  elif [[ "$CREATE_CODE" == "400" ]]; then
    PATCH_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH \
      -H "Authorization: JWT $JWT" \
      -H "Content-Type: application/json" \
      -d '{"is_private":true}' \
      "https://hub.docker.com/v2/repositories/${IMAGE_NAMESPACE}/${IMAGE_NAME}/")
    if [[ "$PATCH_CODE" != "200" && "$PATCH_CODE" != "403" ]]; then
      echo "Repo privacy patch returned HTTP $PATCH_CODE" >&2
      echo "Continuing without enforcing privacy (Docker Hub permissions may be insufficient)." >&2
    fi
  elif [[ "$CREATE_CODE" == "403" ]]; then
    echo "Repo privacy set/create returned HTTP 403; proceeding without changing privacy." >&2
  else
    echo "Repo create request returned HTTP $CREATE_CODE" >&2
    echo "Continuing without enforcing privacy." >&2
  fi
fi

echo "Building image ${IMAGE_BASE}:${VERSION}..."
docker build ${DOCKER_BUILD_ARGS} -t "${IMAGE_BASE}:${VERSION}" "$ROOT_DIR"

if [[ "$PUSH_LATEST" == "true" ]]; then
  docker tag "${IMAGE_BASE}:${VERSION}" "${IMAGE_BASE}:latest"
fi

echo "Pushing ${IMAGE_BASE}:${VERSION}..."
docker push "${IMAGE_BASE}:${VERSION}"

if [[ "$PUSH_LATEST" == "true" ]]; then
  echo "Pushing ${IMAGE_BASE}:latest..."
  docker push "${IMAGE_BASE}:latest"
fi

echo "Done. Published ${IMAGE_BASE}:${VERSION}${PUSH_LATEST:+ and :latest}."
