#!/bin/bash
set -euo pipefail

# GCP Bootstrap Script for Spinner
# Provisions the GCP resources needed to run Spinner with the GCP backend.
# All steps are idempotent — safe to re-run.

usage() {
    cat <<EOF
Usage: $(basename "$0") --project <id> --zone <zone> [options]

Required:
  --project <id>      GCP project ID
  --zone <zone>       GCP zone (e.g. us-central1-a)

Optional:
  --bucket <name>     Bucket name (default: {project}-spinner-state)
  --location <loc>    Bucket location (default: US, or region from zone)
  --sa-name <name>    Service account name (default: spinner-sa)
EOF
    exit 1
}

# --- Parse arguments ---

PROJECT=""
ZONE=""
BUCKET=""
LOCATION=""
SA_NAME="spinner-sa"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --project)  PROJECT="$2";  shift 2 ;;
        --zone)     ZONE="$2";     shift 2 ;;
        --bucket)   BUCKET="$2";   shift 2 ;;
        --location) LOCATION="$2"; shift 2 ;;
        --sa-name)  SA_NAME="$2";  shift 2 ;;
        -h|--help) usage ;;
        *) echo "Unknown option: $1"; usage ;;
    esac
done

if [[ -z "$PROJECT" || -z "$ZONE" ]]; then
    echo "Error: --project and --zone are required"
    usage
fi

if [[ -z "$BUCKET" ]]; then
    BUCKET="${PROJECT}-spinner-state"
fi

# Extract region from zone (e.g. us-central1-a -> us-central1)
REGION="${ZONE%-*}"

# Default bucket location to US multi-region if not specified
# This is more reliable than using the compute region
if [[ -z "$LOCATION" ]]; then
    LOCATION="US"
fi

SA_EMAIL="${SA_NAME}@${PROJECT}.iam.gserviceaccount.com"

# --- Pre-flight checks ---

echo "=== Spinner GCP Bootstrap ==="
echo ""

if ! command -v gcloud &>/dev/null; then
    echo "Error: gcloud CLI is not installed."
    echo "Install it from: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

ACCOUNT=$(gcloud auth list --filter="status:ACTIVE" --format="value(account)" 2>/dev/null || true)
if [[ -z "$ACCOUNT" ]]; then
    echo "Error: No active gcloud account. Run: gcloud auth login"
    exit 1
fi
echo "Authenticated as: $ACCOUNT"

if ! gcloud projects describe "$PROJECT" &>/dev/null; then
    echo "Error: Cannot access project '$PROJECT'. Check the project ID and your permissions."
    exit 1
fi
echo "Project: $PROJECT"
echo "Zone: $ZONE (region: $REGION)"
echo "Bucket: $BUCKET (location: $LOCATION)"
echo "Service account: $SA_EMAIL"
echo ""

# --- Step 1: Enable APIs ---

APIS=(
    compute.googleapis.com
    storage.googleapis.com
    logging.googleapis.com
    monitoring.googleapis.com
)

echo "--- Enabling APIs ---"
for api in "${APIS[@]}"; do
    if gcloud services list --project="$PROJECT" --filter="name:$api" --format="value(name)" 2>/dev/null | grep -q "$api"; then
        echo "  $api (already enabled)"
    else
        echo "  Enabling $api..."
        gcloud services enable "$api" --project="$PROJECT"
    fi
done
echo ""

# --- Step 2: Create GCS bucket ---

echo "--- Creating GCS bucket ---"
if gcloud storage buckets describe "gs://$BUCKET" --project="$PROJECT" &>/dev/null; then
    echo "  gs://$BUCKET (already exists)"
else
    echo "  Creating gs://$BUCKET..."
    gcloud storage buckets create "gs://$BUCKET" \
        --project="$PROJECT" \
        --location="$LOCATION" \
        --uniform-bucket-level-access
fi
echo ""

# --- Step 3: Create service account ---

echo "--- Creating service account ---"
if gcloud iam service-accounts describe "$SA_EMAIL" --project="$PROJECT" &>/dev/null; then
    echo "  $SA_EMAIL (already exists)"
else
    echo "  Creating $SA_EMAIL..."
    gcloud iam service-accounts create "$SA_NAME" \
        --project="$PROJECT" \
        --display-name="Spinner GCP Backend"
fi
echo ""

# --- Step 4: Bind IAM roles ---

PROJECT_ROLES=(
    roles/compute.instanceAdmin.v1
    roles/logging.viewer
    roles/monitoring.viewer
)

echo "--- Binding IAM roles ---"
echo "  Project-level roles:"
for role in "${PROJECT_ROLES[@]}"; do
    EXISTING=$(gcloud projects get-iam-policy "$PROJECT" \
        --flatten="bindings[].members" \
        --filter="bindings.role:$role AND bindings.members:serviceAccount:$SA_EMAIL" \
        --format="value(bindings.role)" 2>/dev/null || true)
    if [[ -n "$EXISTING" ]]; then
        echo "    $role (already bound)"
    else
        echo "    Binding $role..."
        gcloud projects add-iam-policy-binding "$PROJECT" \
            --member="serviceAccount:$SA_EMAIL" \
            --role="$role" \
            --condition=None \
            --quiet >/dev/null
    fi
done

echo "  Bucket-level roles:"
BUCKET_ROLE="roles/storage.objectAdmin"
EXISTING=$(gcloud storage buckets get-iam-policy "gs://$BUCKET" \
    --format=json 2>/dev/null | \
    grep -c "$SA_EMAIL" || true)
if [[ "$EXISTING" -gt 0 ]]; then
    echo "    $BUCKET_ROLE (already bound)"
else
    echo "    Binding $BUCKET_ROLE on gs://$BUCKET..."
    gcloud storage buckets add-iam-policy-binding "gs://$BUCKET" \
        --member="serviceAccount:$SA_EMAIL" \
        --role="$BUCKET_ROLE" \
        --quiet >/dev/null
fi
echo ""

# --- Done: Print export commands ---

echo "=== Bootstrap complete ==="
echo ""
echo "Add these to your shell profile or run before using Spinner:"
echo ""
echo "  # Authenticate (run once):"
echo "  #   gcloud auth application-default login"
echo ""
echo "  export SPINNER_BACKEND=gcp"
echo "  export SPINNER_PROJECT=$PROJECT"
echo "  export SPINNER_ZONE=$ZONE"
echo "  export SPINNER_STATE_BUCKET=$BUCKET"
echo ""
