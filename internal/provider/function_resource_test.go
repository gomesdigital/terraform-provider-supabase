// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/supabase/cli/pkg/api"
	"gopkg.in/h2non/gock.v1"
)

func TestAccFunctionResource(t *testing.T) {
	// Setup mock api
	defer gock.OffAll()

	// Create a temporary directory with a test function file
	tempDir := t.TempDir()
	indexContent := []byte(`Deno.serve(async (req) => { return new Response("Hello World!") })`)
	if err := os.WriteFile(filepath.Join(tempDir, "index.ts"), indexContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Step 1: create (deploy)
	gock.New("https://api.supabase.com").
		Post("/v1/projects/mayuaycdtijbctgqbycg/functions/deploy").
		Reply(http.StatusCreated).
		JSON(api.DeployFunctionResponse{
			Id:        "func-uuid-1234",
			Slug:      "hello-world",
			Name:      "hello-world",
			Status:    api.DeployFunctionResponseStatusACTIVE,
			Version:   1,
			CreatedAt: Ptr(int64(1704067200)),
			UpdatedAt: Ptr(int64(1704067200)),
		})

	// Step 2: read (after create)
	gock.New("https://api.supabase.com").
		Get("/v1/projects/mayuaycdtijbctgqbycg/functions/hello-world").
		Reply(http.StatusOK).
		JSON(api.FunctionSlugResponse{
			Id:        "func-uuid-1234",
			Slug:      "hello-world",
			Name:      "hello-world",
			Status:    api.FunctionSlugResponseStatusACTIVE,
			Version:   1,
			CreatedAt: 1704067200,
			UpdatedAt: 1704067200,
			VerifyJwt: Ptr(true),
		})

	// Step 3: read (before update)
	gock.New("https://api.supabase.com").
		Get("/v1/projects/mayuaycdtijbctgqbycg/functions/hello-world").
		Reply(http.StatusOK).
		JSON(api.FunctionSlugResponse{
			Id:        "func-uuid-1234",
			Slug:      "hello-world",
			Name:      "hello-world",
			Status:    api.FunctionSlugResponseStatusACTIVE,
			Version:   1,
			CreatedAt: 1704067200,
			UpdatedAt: 1704067200,
			VerifyJwt: Ptr(true),
		})

	// Step 4: update (uses deploy endpoint for proper ESZIP bundling)
	gock.New("https://api.supabase.com").
		Post("/v1/projects/mayuaycdtijbctgqbycg/functions/deploy").
		Reply(http.StatusCreated).
		JSON(api.DeployFunctionResponse{
			Id:        "func-uuid-1234",
			Slug:      "hello-world",
			Name:      "hello-world",
			Status:    api.DeployFunctionResponseStatusACTIVE,
			Version:   2,
			CreatedAt: Ptr(int64(1704067200)),
			UpdatedAt: Ptr(int64(1704067300)),
			VerifyJwt: Ptr(false),
		})

	// Step 5: read (after update)
	gock.New("https://api.supabase.com").
		Get("/v1/projects/mayuaycdtijbctgqbycg/functions/hello-world").
		Reply(http.StatusOK).
		JSON(api.FunctionSlugResponse{
			Id:        "func-uuid-1234",
			Slug:      "hello-world",
			Name:      "hello-world",
			Status:    api.FunctionSlugResponseStatusACTIVE,
			Version:   2,
			CreatedAt: 1704067200,
			UpdatedAt: 1704067300,
			VerifyJwt: Ptr(false),
		})

	// Step 6: read (refresh before destroy)
	gock.New("https://api.supabase.com").
		Get("/v1/projects/mayuaycdtijbctgqbycg/functions/hello-world").
		Reply(http.StatusOK).
		JSON(api.FunctionSlugResponse{
			Id:        "func-uuid-1234",
			Slug:      "hello-world",
			Name:      "hello-world",
			Status:    api.FunctionSlugResponseStatusACTIVE,
			Version:   2,
			CreatedAt: 1704067200,
			UpdatedAt: 1704067300,
			VerifyJwt: Ptr(false),
		})

	// Step 7: delete
	gock.New("https://api.supabase.com").
		Delete("/v1/projects/mayuaycdtijbctgqbycg/functions/hello-world").
		Reply(http.StatusOK).
		JSON(map[string]interface{}{})

	// Run test
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccFunctionResourceConfig(tempDir, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supabase_function.test", "id", "func-uuid-1234"),
					resource.TestCheckResourceAttr("supabase_function.test", "slug", "hello-world"),
					resource.TestCheckResourceAttr("supabase_function.test", "name", "hello-world"),
					resource.TestCheckResourceAttr("supabase_function.test", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("supabase_function.test", "version", "1"),
					resource.TestCheckResourceAttr("supabase_function.test", "verify_jwt", "true"),
				),
			},
			// Update testing
			{
				Config: testAccFunctionResourceConfig(tempDir, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supabase_function.test", "id", "func-uuid-1234"),
					resource.TestCheckResourceAttr("supabase_function.test", "version", "2"),
					resource.TestCheckResourceAttr("supabase_function.test", "verify_jwt", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccFunctionResourceConfig(sourceDir string, verifyJwt bool) string {
	verifyJwtStr := "false"
	if verifyJwt {
		verifyJwtStr = "true"
	}
	return `
resource "supabase_function" "test" {
  project_ref     = "mayuaycdtijbctgqbycg"
  slug            = "hello-world"
  entrypoint_path = "index.ts"
  source_dir      = "` + sourceDir + `"
  verify_jwt      = ` + verifyJwtStr + `
}
`
}
