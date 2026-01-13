#!/bin/bash

# =============================================================================
# Supabase Terraform Provider - Edge Functions E2E Demo (Automated)
# =============================================================================
#
# This is the non-interactive version of the demo script, suitable for CI/CD.
#
# This script demonstrates all Edge Function features:
#   1. Deploy (Create) a function
#   2. Read (Retrieve) a function
#   3. Retrieve function body
#   4. Update a function
#   5. Delete a function
#
# Usage:
#   SUPABASE_TOKEN=<token> PROJECT_REF=<ref> ./demo-auto.sh
#
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Configuration - Set via environment variables
SUPABASE_TOKEN="${SUPABASE_TOKEN:-sbp_32a5c4e7436d8c61d13b8eefe0165a7605ae7f0b}"
PROJECT_REF="${PROJECT_REF:-tyrtejildjzsnpzlpucq}"

# Validate configuration
if [ -z "${SUPABASE_TOKEN}" ]; then
    echo -e "${RED}Error: SUPABASE_TOKEN environment variable is required${NC}"
    exit 1
fi

if [ -z "${PROJECT_REF}" ]; then
    echo -e "${RED}Error: PROJECT_REF environment variable is required${NC}"
    exit 1
fi

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEMO_DIR="${SCRIPT_DIR}/demo-auto"
FUNCTIONS_DIR="${SCRIPT_DIR}/../examples/resources/supabase_function/functions/hello-world"

# Function slug for the demo (unique per run)
FUNCTION_SLUG="tf-auto-$(date +%s)"

# =============================================================================
# Helper Functions
# =============================================================================

print_header() {
    echo ""
    echo -e "${BLUE}=============================================================================${NC}"
    echo -e "${BLUE}${BOLD} $1${NC}"
    echo -e "${BLUE}=============================================================================${NC}"
    echo ""
}

print_step() {
    echo -e "${CYAN}>>> $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${BOLD}$1${NC}"
}

cleanup() {
    print_header "CLEANUP"
    print_step "Cleaning up demo directory..."
    rm -rf "${DEMO_DIR}"
    print_success "Demo directory cleaned up"
}

# Set up trap to cleanup on exit
trap cleanup EXIT

# =============================================================================
# Setup
# =============================================================================

print_header "SETUP"

print_step "Creating demo directory..."
mkdir -p "${DEMO_DIR}"
cd "${DEMO_DIR}"

print_step "Setting up Terraform CLI configuration..."
export TF_CLI_CONFIG_FILE="${SCRIPT_DIR}/terraform.tfrc"

print_success "Environment configured"
print_info "  Token: ${SUPABASE_TOKEN:0:10}..."
print_info "  Project Ref: ${PROJECT_REF}"
print_info "  Function Slug: ${FUNCTION_SLUG}"

# =============================================================================
# STEP 1: Deploy (Create) a Function
# =============================================================================

print_header "STEP 1: DEPLOY (CREATE) A FUNCTION"

print_step "Creating Terraform configuration..."

cat > main.tf << EOF
terraform {
  required_providers {
    supabase = {
      source = "supabase/supabase"
    }
  }
}

provider "supabase" {
  access_token = var.supabase_access_token
}

variable "supabase_access_token" {
  type      = string
  sensitive = true
}

variable "project_ref" {
  type = string
}

resource "supabase_function" "demo" {
  project_ref     = var.project_ref
  slug            = "${FUNCTION_SLUG}"
  name            = "Demo Function"
  entrypoint_path = "index.ts"
  source_dir      = "${FUNCTIONS_DIR}"
  verify_jwt      = false
}

output "function_id" {
  value = supabase_function.demo.id
}

output "function_slug" {
  value = supabase_function.demo.slug
}

output "function_name" {
  value = supabase_function.demo.name
}

output "function_status" {
  value = supabase_function.demo.status
}

output "function_version" {
  value = supabase_function.demo.version
}
EOF

print_step "Running terraform plan..."
terraform plan \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_step "Running terraform apply..."
terraform apply -auto-approve \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_success "Function deployed!"

# =============================================================================
# STEP 2: Read (Retrieve) a Function
# =============================================================================

print_header "STEP 2: READ (RETRIEVE) A FUNCTION"

print_step "Running terraform refresh..."
terraform refresh \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_step "Showing current state..."
terraform show

print_success "Function state retrieved!"

# =============================================================================
# STEP 3: Retrieve Function Body
# =============================================================================

print_header "STEP 3: RETRIEVE FUNCTION BODY"

print_step "Adding data source for function body..."

cat >> main.tf << EOF

data "supabase_function_body" "demo" {
  project_ref = var.project_ref
  slug        = supabase_function.demo.slug
}

output "function_body" {
  value = data.supabase_function_body.demo.body
}
EOF

print_step "Running terraform apply..."
terraform apply -auto-approve \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_step "Function body output:"
terraform output function_body

print_success "Function body retrieved!"

# =============================================================================
# STEP 4: Update a Function
# =============================================================================

print_header "STEP 4: UPDATE A FUNCTION"

print_step "Creating updated function source..."

mkdir -p "${DEMO_DIR}/updated-function"
cat > "${DEMO_DIR}/updated-function/index.ts" << 'EOF'
Deno.serve(async (req) => {
  return new Response(
    JSON.stringify({
      message: "Hello from Terraform - UPDATED!",
      version: 2
    }),
    { headers: { "Content-Type": "application/json" } }
  )
})
EOF

print_step "Updating Terraform configuration..."

cat > main.tf << EOF
terraform {
  required_providers {
    supabase = {
      source = "supabase/supabase"
    }
  }
}

provider "supabase" {
  access_token = var.supabase_access_token
}

variable "supabase_access_token" {
  type      = string
  sensitive = true
}

variable "project_ref" {
  type = string
}

resource "supabase_function" "demo" {
  project_ref     = var.project_ref
  slug            = "${FUNCTION_SLUG}"
  name            = "Demo Function v2"
  entrypoint_path = "index.ts"
  source_dir      = "${DEMO_DIR}/updated-function"
  verify_jwt      = true
}

data "supabase_function_body" "demo" {
  project_ref = var.project_ref
  slug        = supabase_function.demo.slug
}

output "function_id" {
  value = supabase_function.demo.id
}

output "function_slug" {
  value = supabase_function.demo.slug
}

output "function_name" {
  value = supabase_function.demo.name
}

output "function_status" {
  value = supabase_function.demo.status
}

output "function_version" {
  value = supabase_function.demo.version
}

output "function_verify_jwt" {
  value = supabase_function.demo.verify_jwt
}

output "function_body" {
  value = data.supabase_function_body.demo.body
}
EOF

print_step "Running terraform plan..."
terraform plan \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_step "Running terraform apply..."
terraform apply -auto-approve \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_step "Verifying the update..."
terraform show

print_success "Function updated!"

# =============================================================================
# STEP 5: Delete a Function
# =============================================================================

print_header "STEP 5: DELETE A FUNCTION"

print_step "Running terraform plan -destroy..."
terraform plan -destroy \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_step "Running terraform destroy..."
terraform destroy -auto-approve \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_success "Function deleted!"

# =============================================================================
# SUMMARY
# =============================================================================

print_header "DEMO COMPLETE"

echo -e "${GREEN}${BOLD}All Edge Function features demonstrated successfully!${NC}"
echo ""
print_info "Features demonstrated:"
echo "  ✓ Deploy (Create) a function"
echo "  ✓ Read (Retrieve) a function"
echo "  ✓ Retrieve function body"
echo "  ✓ Update a function"
echo "  ✓ Delete a function"
echo ""
print_success "Demo completed successfully!"

exit 0
