Deno.serve(async (req) => {
  return new Response(
    JSON.stringify({ message: "Hello from Terraform!" }),
    { headers: { "Content-Type": "application/json" } }
  )
})
