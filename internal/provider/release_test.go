package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccReleaseBasic(t *testing.T) {
	testAccPreCheck(t)

	releaseName := testAccReleaseName("basic")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReleaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccReleaseConfig(releaseName, "hello"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "name", releaseName),
					resource.TestCheckResourceAttr("helm_release.test", "namespace", "default"),
					resource.TestCheckResourceAttr("helm_release.test", "status", "deployed"),
					resource.TestCheckResourceAttrSet("helm_release.test", "manifests"),
					testAccCheckHelmReleaseManifestContains(releaseName, "default", "message: hello"),
				),
			},
		},
	})
}

func TestAccReleaseUpdate(t *testing.T) {
	testAccPreCheck(t)

	releaseName := testAccReleaseName("update")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReleaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccReleaseConfig(releaseName, "hello"),
				Check:  testAccCheckHelmReleaseManifestContains(releaseName, "default", "message: hello"),
			},
			{
				Config: testAccReleaseConfig(releaseName, "updated"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "status", "deployed"),
					testAccCheckHelmReleaseManifestContains(releaseName, "default", "message: updated"),
				),
			},
		},
	})
}

func TestAccReleaseNoop(t *testing.T) {
	testAccPreCheck(t)

	releaseName := testAccReleaseName("noop")
	config := testAccReleaseConfig(releaseName, "steady")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReleaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckHelmReleaseManifestContains(releaseName, "default", "message: steady"),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

func TestAccReleaseDestroy(t *testing.T) {
	testAccPreCheck(t)

	releaseName := testAccReleaseName("destroy")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReleaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccReleaseConfig(releaseName, "bye"),
				Check:  testAccCheckHelmReleaseManifestContains(releaseName, "default", "message: bye"),
			},
		},
	})
}
