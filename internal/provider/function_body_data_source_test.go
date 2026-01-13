// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gopkg.in/h2non/gock.v1"
)

func TestAccFunctionBodyDataSource(t *testing.T) {
	functionBody := `Deno.serve(async (req) => { return new Response("Hello World!") })`

	// Setup mock api
	defer gock.OffAll()
	gock.New("https://api.supabase.com").
		Get("/v1/projects/mayuaycdtijbctgqbycg/functions/hello-world/body").
		Persist().
		Reply(http.StatusOK).
		BodyString(functionBody)

	// Run test
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionBodyDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.supabase_function_body.test", "project_ref", "mayuaycdtijbctgqbycg"),
					resource.TestCheckResourceAttr("data.supabase_function_body.test", "slug", "hello-world"),
					resource.TestCheckResourceAttr("data.supabase_function_body.test", "body", functionBody),
				),
			},
		},
	})
}

const testAccFunctionBodyDataSourceConfig = `
data "supabase_function_body" "test" {
  project_ref = "mayuaycdtijbctgqbycg"
  slug        = "hello-world"
}
`
