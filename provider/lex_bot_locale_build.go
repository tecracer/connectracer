// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lexmodelsv2"
	lexv2types "github.com/aws/aws-sdk-go-v2/service/lexmodelsv2/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func buildBotLocaleAndWait(ctx context.Context, client *lexmodelsv2.Client, botID, botVersion, localeID string) error {
	tflog.Debug(ctx, "Building Lex V2 bot locale", map[string]interface{}{
		"bot_id":      botID,
		"bot_version": botVersion,
		"locale_id":   localeID,
	})

	_, err := client.BuildBotLocale(ctx, &lexmodelsv2.BuildBotLocaleInput{
		BotId:      aws.String(botID),
		BotVersion: aws.String(botVersion),
		LocaleId:   aws.String(localeID),
	})
	if err != nil {
		return fmt.Errorf("starting bot locale build: %w", err)
	}

	return waitForBotLocaleBuilt(ctx, client, botID, botVersion, localeID)
}

func waitForBotLocaleBuilt(ctx context.Context, client *lexmodelsv2.Client, botID, botVersion, localeID string) error {
	const (
		maxAttempts  = 60
		pollInterval = 5 * time.Second
	)

	for i := range maxAttempts {
		output, err := client.DescribeBotLocale(ctx, &lexmodelsv2.DescribeBotLocaleInput{
			BotId:      aws.String(botID),
			BotVersion: aws.String(botVersion),
			LocaleId:   aws.String(localeID),
		})
		if err != nil {
			return fmt.Errorf("polling bot locale build status: %w", err)
		}

		switch output.BotLocaleStatus {
		case lexv2types.BotLocaleStatusBuilt, lexv2types.BotLocaleStatusReadyExpressTesting:
			return nil
		case lexv2types.BotLocaleStatusFailed:
			reason := "unknown"
			if len(output.FailureReasons) > 0 {
				reason = output.FailureReasons[0]
			}
			return fmt.Errorf("bot locale build failed: %s", reason)
		}

		tflog.Debug(ctx, "Waiting for bot locale build", map[string]interface{}{
			"attempt": i + 1,
			"status":  string(output.BotLocaleStatus),
		})
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("bot locale %q did not reach Built state after %d attempts", localeID, maxAttempts)
}
