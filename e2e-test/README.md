# Supabase Terraform Provider - E2E Tests

This directory contains end-to-end tests and demo scripts for the Supabase Terraform Provider.

## Demo Scripts

### Interactive Demo (`demo.sh`)

An interactive demo that walks through all Edge Function features step-by-step, pausing for user input between each operation. This is ideal for demonstrations and presentations.

```bash
# Set your credentials
export SUPABASE_TOKEN="your-access-token"
export PROJECT_REF="your-project-ref"

# Run the interactive demo
./demo.sh
```

### Automated Demo (`demo-auto.sh`)

A non-interactive version suitable for CI/CD pipelines or quick testing.

```bash
# Run with environment variables
SUPABASE_TOKEN="your-access-token" PROJECT_REF="your-project-ref" ./demo-auto.sh
```

## Features Demonstrated

Both scripts demonstrate the following Edge Function features:

1. **Deploy (Create)** - Deploy a new Edge Function using the `/v1/projects/{ref}/functions/deploy` endpoint
2. **Read (Retrieve)** - Read function metadata using `terraform refresh` and `terraform show`
3. **Retrieve Function Body** - Use the `supabase_function_body` data source to fetch deployed source code
4. **Update** - Update function properties (name, verify_jwt) and source code
5. **Delete** - Remove the function using `terraform destroy`

## Prerequisites

- Terraform installed
- The provider built (`go build -o terraform-provider-supabase`)
- Valid Supabase API access token
- A Supabase project reference ID

## Configuration

The scripts use the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SUPABASE_TOKEN` | Supabase API access token | (required) |
| `PROJECT_REF` | Supabase project reference ID | (required) |

## Directory Structure

```
e2e-test/
├── README.md           # This file
├── demo.sh             # Interactive demo script
├── demo-auto.sh        # Automated demo script
├── main.tf             # Basic test configuration
└── terraform.tfrc      # Terraform CLI configuration for local provider
```

## Terraform CLI Configuration

The `terraform.tfrc` file configures Terraform to use the locally built provider:

```hcl
provider_installation {
  dev_overrides {
    "supabase/supabase" = "/testbed"
  }
  direct {}
}
```

This is automatically used by the demo scripts via `TF_CLI_CONFIG_FILE`.

## Cleanup

Both scripts automatically clean up created resources and temporary files on exit, even if an error occurs (using shell traps).
