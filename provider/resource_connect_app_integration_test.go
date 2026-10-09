// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appintegrations/types"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAppIntegrationAuthConfigRoundTrip(t *testing.T) {
	const secretArn = "arn:aws:secretsmanager:eu-central-1:111122223333:secret:a2a-api-key-AbCdEf"

	model := &AppIntegrationAuthConfigModel{
		AuthType:                     frameworktypes.StringValue("API_KEY"),
		CredentialProviderIdentifier: frameworktypes.StringValue(secretArn),
	}

	expanded := expandAppIntegrationAuthConfig(model)
	if expanded.AuthType != types.AuthTypeApiKey || aws.ToString(expanded.CredentialProviderIdentifier) != secretArn {
		t.Fatalf("expandAppIntegrationAuthConfig() = %+v", expanded)
	}

	flattened := flattenAppIntegrationAuthConfig(expanded)
	if !flattened.AuthType.Equal(model.AuthType) || !flattened.CredentialProviderIdentifier.Equal(model.CredentialProviderIdentifier) {
		t.Errorf("flattenAppIntegrationAuthConfig() = %+v, want %+v", flattened, model)
	}
}

func TestAppIntegrationWithoutAuthConfigSendsNone(t *testing.T) {
	if expandAppIntegrationAuthConfig(nil) != nil {
		t.Error("an application without auth_config must not send one")
	}
	if flattenAppIntegrationAuthConfig(nil) != nil {
		t.Error("an application AWS returns without AuthConfig must read back as null")
	}
}
