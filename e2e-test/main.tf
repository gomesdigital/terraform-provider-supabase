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

resource "supabase_function" "hello" {
  project_ref     = var.project_ref
  slug            = "tf-hello-world"
  entrypoint_path = "index.ts"
  source_dir      = "${path.module}/../examples/resources/supabase_function/functions/hello-world"
  verify_jwt      = false
}

output "function_id" {
  value = supabase_function.hello.id
}

output "function_status" {
  value = supabase_function.hello.status
}

output "function_version" {
  value = supabase_function.hello.version
}
