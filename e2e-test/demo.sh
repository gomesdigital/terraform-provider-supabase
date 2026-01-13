#!/bin/bash

# =============================================================================
# Supabase Terraform Provider - Edge Functions E2E Demo
# =============================================================================
#
# This script demonstrates all Edge Function features that have been implemented:
#   1. Deploy (Create) a function
#   2. Read (Retrieve) a function
#   3. Retrieve function body
#   4. Update a function
#   5. Delete a function
#
# Usage: ./demo.sh
#
# Prerequisites:
#   - Terraform installed
#   - The provider built and available
#   - Valid Supabase API token and project reference
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

# Configuration - Update these values as needed
SUPABASE_TOKEN="${SUPABASE_TOKEN:-sbp_32a5c4e7436d8c61d13b8eefe0165a7605ae7f0b}"
PROJECT_REF="${PROJECT_REF:-tyrtejildjzsnpzlpucq}"

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEMO_DIR="${SCRIPT_DIR}/demo"
FUNCTIONS_DIR="${SCRIPT_DIR}/../examples/resources/supabase_function/functions/hello-world"

# Function slug for the demo
FUNCTION_SLUG="tf-demo-$(date +%s)"

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

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${BOLD}$1${NC}"
}

wait_for_user() {
    echo ""
    echo -e "${YELLOW}Press Enter to continue...${NC}"
    read -r
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
print_info "  Demo Dir: ${DEMO_DIR}"

wait_for_user

# =============================================================================
# STEP 1: Deploy (Create) a Function
# =============================================================================

print_header "STEP 1: DEPLOY (CREATE) A FUNCTION"

print_step "Creating Terraform configuration for function deployment..."

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

# Deploy an Edge Function
resource "supabase_function" "demo" {
  project_ref     = var.project_ref
  slug            = "${FUNCTION_SLUG}"
  name            = "Demo Function"
  entrypoint_path = "index.ts"
  source_dir      = "${FUNCTIONS_DIR}"
  verify_jwt      = false
}

# Outputs to show created function details
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

print_success "Terraform configuration created"
echo ""
print_info "Configuration contents:"
echo "----------------------------------------"
cat main.tf
echo "----------------------------------------"

wait_for_user

print_step "Running 'terraform plan' to preview the deployment..."
echo ""
terraform plan \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

wait_for_user

print_step "Running 'terraform apply' to deploy the function..."
echo ""
terraform apply -auto-approve \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_success "Function deployed successfully!"

wait_for_user

# =============================================================================
# STEP 2: Read (Retrieve) a Function
# =============================================================================

print_header "STEP 2: READ (RETRIEVE) A FUNCTION"

print_step "Running 'terraform refresh' to read the current state of the function..."
echo ""
terraform refresh \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_step "Displaying current state with 'terraform show'..."
echo ""
terraform show

print_success "Function state retrieved successfully!"

wait_for_user

# =============================================================================
# STEP 3: Retrieve Function Body (Data Source)
# =============================================================================

print_header "STEP 3: RETRIEVE FUNCTION BODY (DATA SOURCE)"

print_step "Adding data source to retrieve function body..."

cat >> main.tf << EOF

# Data source to retrieve the function's body/source code
data "supabase_function_body" "demo" {
  project_ref = var.project_ref
  slug        = supabase_function.demo.slug
}

output "function_body" {
  value = data.supabase_function_body.demo.body
}
EOF

print_success "Data source configuration added"
echo ""
print_info "Updated configuration:"
echo "----------------------------------------"
cat main.tf
echo "----------------------------------------"

wait_for_user

print_step "Running 'terraform plan' to show the data source..."
echo ""
terraform plan \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

wait_for_user

print_step "Running 'terraform apply' to fetch the function body..."
echo ""
terraform apply -auto-approve \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_step "Displaying function body output..."
echo ""
terraform output function_body

print_success "Function body retrieved successfully!"

wait_for_user

# =============================================================================
# STEP 4: Update a Function
# =============================================================================

print_header "STEP 4: UPDATE A FUNCTION"

print_step "Creating updated function source code..."

mkdir -p "${DEMO_DIR}/updated-function"
cat > "${DEMO_DIR}/updated-function/index.ts" << 'EOF'
Deno.serve(async (req) => {
  const message = "Hello from Terraform - UPDATED!";
  const timestamp = new Date().toISOString();

  return new Response(
    JSON.stringify({
      message: message,
      updated_at: timestamp,
      version: 2
    }),
    { headers: { "Content-Type": "application/json" } }
  )
})
EOF

print_success "Updated source code created"
echo ""
print_info "New source code:"
echo "----------------------------------------"
cat "${DEMO_DIR}/updated-function/index.ts"
echo "----------------------------------------"

wait_for_user

print_step "Updating Terraform configuration to use new source..."

# Rewrite main.tf with updated source_dir, name, and verify_jwt
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

# Deploy an Edge Function (UPDATED)
resource "supabase_function" "demo" {
  project_ref     = var.project_ref
  slug            = "${FUNCTION_SLUG}"
  name            = "Demo Function v2"  # Updated name
  entrypoint_path = "index.ts"
  source_dir      = "${DEMO_DIR}/updated-function"  # Updated source
  verify_jwt      = true  # Changed from false to true
}

# Data source to retrieve the function's body/source code
data "supabase_function_body" "demo" {
  project_ref = var.project_ref
  slug        = supabase_function.demo.slug
}

# Outputs
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

print_success "Configuration updated"
echo ""
print_info "Changes made:"
echo "  - name: 'Demo Function' -> 'Demo Function v2'"
echo "  - source_dir: original -> updated-function"
echo "  - verify_jwt: false -> true"

wait_for_user

print_step "Running 'terraform plan' to preview the update..."
echo ""
terraform plan \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

wait_for_user

print_step "Running 'terraform apply' to update the function..."
echo ""
terraform apply -auto-approve \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_success "Function updated successfully!"

print_step "Verifying the update with terraform show..."
echo ""
terraform show

wait_for_user

# =============================================================================
# STEP 5: Delete a Function
# =============================================================================

print_header "STEP 5: DELETE A FUNCTION"

print_step "Running 'terraform plan -destroy' to preview deletion..."
echo ""
terraform plan -destroy \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

wait_for_user

print_step "Running 'terraform destroy' to delete the function..."
echo ""
terraform destroy -auto-approve \
  -var="supabase_access_token=${SUPABASE_TOKEN}" \
  -var="project_ref=${PROJECT_REF}"

print_success "Function deleted successfully!"

# =============================================================================
# SUMMARY
# =============================================================================

print_header "DEMO COMPLETE"

echo -e "${GREEN}${BOLD}All Edge Function features have been demonstrated successfully!${NC}"
echo ""
print_info "Features demonstrated:"
echo "  ✓ Deploy (Create) a function - Using the /deploy API endpoint"
echo "  ✓ Read (Retrieve) a function - Using terraform refresh/show"
echo "  ✓ Retrieve function body     - Using supabase_function_body data source"
echo "  ✓ Update a function          - Modifying name, source, and verify_jwt"
echo "  ✓ Delete a function          - Using terraform destroy"
echo ""
print_info "Function slug used: ${FUNCTION_SLUG}"
echo ""
print_success "Demo completed and resources cleaned up!"
