// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"time"
)

// eventualConsistencyRetryDelays backs off while a dependent AWS API call fails
// because a resource created earlier in the same apply hasn't propagated yet —
// e.g. AssociateSecurityProfiles not yet recognizing a security profile ID
// returned moments earlier by CreateSecurityProfile, or UpdateSecurityProfile
// not yet recognizing a flow module ID returned moments earlier by
// CreateContactFlowModule. This is a real, repeatedly observed gap between
// these newer Connect APIs, not a rare fluke, since "create X then immediately
// reference X elsewhere" is the common case for these resources.
var eventualConsistencyRetryDelays = []time.Duration{
	2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second,
}

// retryOnEventualConsistency calls op repeatedly while shouldRetry(err) is
// true for the returned error, backing off per eventualConsistencyRetryDelays,
// until op succeeds, shouldRetry returns false, attempts are exhausted, or ctx
// is done.
func retryOnEventualConsistency(ctx context.Context, shouldRetry func(error) bool, op func() error) error {
	for attempt := 0; ; attempt++ {
		err := op()
		if err == nil {
			return nil
		}
		if !shouldRetry(err) || attempt >= len(eventualConsistencyRetryDelays) {
			return err
		}
		select {
		case <-time.After(eventualConsistencyRetryDelays[attempt]):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
