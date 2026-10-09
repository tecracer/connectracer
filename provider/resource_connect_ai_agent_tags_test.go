// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import "testing"

func TestUnqualifiedAIAgentArn(t *testing.T) {
	t.Parallel()

	base := "arn:aws:wisdom:eu-central-1:123456789012:ai-agent/assistant-id/agent-id"
	for _, arn := range []string{base, base + ":$LATEST", base + ":$SAVED", base + ":3"} {
		if got := unqualifiedAIAgentArn(arn); got != base {
			t.Errorf("unqualifiedAIAgentArn(%q) = %q, want %q", arn, got, base)
		}
	}
}
