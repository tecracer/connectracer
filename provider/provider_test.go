// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"connectracer": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside connectracer.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"connectracer": providerserver.NewProtocol6WithError(New("test")()),
	"echo":         echoprovider.NewProviderServer(),
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests require TF_ACC=1")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Skipf("skipping acceptance test: unable to load AWS config: %v", err)
	}

	if _, err := cfg.Credentials.Retrieve(ctx); err != nil {
		t.Skipf("skipping acceptance test: no AWS credentials available: %v", err)
	}
}
