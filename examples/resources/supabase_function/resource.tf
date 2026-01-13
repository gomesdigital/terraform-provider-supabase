resource "supabase_function" "hello" {
  project_ref     = "mayuaycdtijbctgqbycg"
  slug            = "hello-world"
  entrypoint_path = "index.ts"
  source_dir      = "${path.module}/functions/hello-world"
  verify_jwt      = true
}
