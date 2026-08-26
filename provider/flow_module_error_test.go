// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
)

func TestFlowModuleErrorDetail(t *testing.T) {
	t.Parallel()

	t.Run("plain error falls back to Error()", func(t *testing.T) {
		t.Parallel()
		err := errors.New("boom")
		if got := flowModuleErrorDetail(err); got != "boom" {
			t.Fatalf("flowModuleErrorDetail() = %q, want %q", got, "boom")
		}
	})

	t.Run("InvalidContactFlowModuleException with no problems falls back to Error()", func(t *testing.T) {
		t.Parallel()
		err := &types.InvalidContactFlowModuleException{Message: aws.String("bad module")}
		got := flowModuleErrorDetail(err)
		if got != err.Error() {
			t.Fatalf("flowModuleErrorDetail() = %q, want %q", got, err.Error())
		}
	})

	t.Run("InvalidContactFlowModuleException surfaces Problems", func(t *testing.T) {
		t.Parallel()
		err := &types.InvalidContactFlowModuleException{
			Problems: []types.ProblemDetail{
				{Message: aws.String("Actions[0].Parameters.LambdaFunctionARN is invalid")},
				{Message: aws.String("Settings is invalid")},
			},
		}
		got := flowModuleErrorDetail(err)
		if !strings.Contains(got, "Actions[0].Parameters.LambdaFunctionARN is invalid") {
			t.Fatalf("flowModuleErrorDetail() = %q, want it to contain the first problem", got)
		}
		if !strings.Contains(got, "Settings is invalid") {
			t.Fatalf("flowModuleErrorDetail() = %q, want it to contain the second problem", got)
		}
	})

	t.Run("wrapped InvalidContactFlowModuleException is still unwrapped via errors.As", func(t *testing.T) {
		t.Parallel()
		inner := &types.InvalidContactFlowModuleException{
			Problems: []types.ProblemDetail{{Message: aws.String("nested problem")}},
		}
		wrapped := errorsJoinFmt(inner)
		got := flowModuleErrorDetail(wrapped)
		if !strings.Contains(got, "nested problem") {
			t.Fatalf("flowModuleErrorDetail() = %q, want it to contain the wrapped problem", got)
		}
	})
}

// errorsJoinFmt wraps err the way fmt.Errorf("...: %w", err) would, without
// pulling in fmt just for this test helper's own error text.
func errorsJoinFmt(err error) error {
	return &wrappedError{err}
}

type wrappedError struct{ inner error }

func (w *wrappedError) Error() string { return "operation failed: " + w.inner.Error() }
func (w *wrappedError) Unwrap() error { return w.inner }
