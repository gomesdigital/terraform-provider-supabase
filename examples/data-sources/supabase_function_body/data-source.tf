data "supabase_function_body" "example" {
  project_ref = "mayuaycdtijbctgqbycg"
  slug        = "hello-world"
}

output "function_source" {
  value = data.supabase_function_body.example.body
}
