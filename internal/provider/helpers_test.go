package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	terraformstate "github.com/hashicorp/terraform-plugin-testing/terraform"

	"helm.sh/helm/v4/pkg/action"
	releasepkg "helm.sh/helm/v4/pkg/release"
	helmrelease "helm.sh/helm/v4/pkg/release/v1"
	"helm.sh/helm/v4/pkg/storage/driver"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"helm": providerserver.NewProtocol6WithError(New("test", "test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests skipped unless TF_ACC is set")
	}

	kubeConfigPath := os.Getenv("KUBE_CONFIG_PATH")
	if kubeConfigPath == "" {
		t.Fatal("KUBE_CONFIG_PATH must be set for acceptance tests")
	}

	if _, err := os.Stat(kubeConfigPath); err != nil {
		t.Fatalf("KUBE_CONFIG_PATH %q is not readable: %v", kubeConfigPath, err)
	}
}

func testAccReleaseConfig(name, message string) string {
	chartPath, err := filepath.Abs(filepath.Join("..", "testdata", "charts", "test-release"))
	if err != nil {
		panic(err)
	}

	chartPath = filepath.ToSlash(chartPath)

	return fmt.Sprintf(`
provider "helm" {
  kubernetes {}
}

resource "helm_release" "test" {
  chart     = %q
  name      = %q
  namespace = "default"
  version   = "0.1.0"
  values = {
    message = %q
  }
}
`, chartPath, name, message)
}

func testAccReleaseName(suffix string) string {
	return fmt.Sprintf("acc-%s-%d", suffix, time.Now().UnixNano())
}

func testAccCheckHelmReleaseManifestContains(name, namespace, expected string) resource.TestCheckFunc {
	return func(_ *terraformstate.State) error {
		rel, err := testAccGetRelease(name, namespace)
		if err != nil {
			return err
		}

		if !strings.Contains(rel.Manifest, expected) {
			return fmt.Errorf("expected live manifest for %s/%s to contain %q", namespace, name, expected)
		}

		return nil
	}
}

func testAccCheckReleaseDestroy(state *terraformstate.State) error {
	for _, rs := range state.RootModule().Resources {
		if rs.Type != "helm_release" {
			continue
		}

		if err := testAccEnsureReleaseMissing(rs.Primary.Attributes["name"], rs.Primary.Attributes["namespace"]); err != nil {
			return err
		}
	}

	return nil
}

func testAccEnsureReleaseMissing(name, namespace string) error {
	_, err := testAccGetRelease(name, namespace)
	if err == nil {
		return fmt.Errorf("expected release %s/%s to be destroyed", namespace, name)
	}

	if strings.Contains(err.Error(), driver.ErrReleaseNotFound.Error()) {
		return nil
	}

	return err
}

func testAccGetRelease(name, namespace string) (*helmrelease.Release, error) {
	ctx := context.Background()
	client, diags := getHelmClient(ctx, HelmProviderModel{Kubernetes: &KubernetesConfigModel{}})
	if diags.HasError() {
		return nil, fmt.Errorf("build helm test client: %v", diags.Errors())
	}

	actionConfig, err := client.GetActionConfigE(namespace)
	if err != nil {
		return nil, err
	}

	getAction := action.NewGet(actionConfig)
	rel, err := getAction.Run(name)
	if err != nil {
		return nil, err
	}

	return releaseResultToRelease(rel)
}

func releaseResultToRelease(rel releasepkg.Releaser) (*helmrelease.Release, error) {
	if release, ok := rel.(*helmrelease.Release); ok {
		return release, nil
	}

	if release, ok := rel.(helmrelease.Release); ok {
		return &release, nil
	}

	return nil, fmt.Errorf("expected *release.Release, got %T", rel)
}
