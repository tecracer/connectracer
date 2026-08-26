// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
)

// ServerSideEncryptionConfigurationModel describes server-side encryption configuration.
type ServerSideEncryptionConfigurationModel struct {
	KmsKeyId frameworktypes.String `tfsdk:"kms_key_id"`
}

func resolveVersionNumber(current, prior frameworktypes.Int64) frameworktypes.Int64 {
	if !current.IsNull() && !current.IsUnknown() {
		return current
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		return prior
	}
	return frameworktypes.Int64Null()
}
