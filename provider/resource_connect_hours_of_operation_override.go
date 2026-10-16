// Copyright tecRacer Group 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ConnectHoursOfOperationOverrideResource{}
var _ resource.ResourceWithImportState = &ConnectHoursOfOperationOverrideResource{}

func NewConnectHoursOfOperationOverrideResource() resource.Resource {
	return &ConnectHoursOfOperationOverrideResource{}
}

type ConnectHoursOfOperationOverrideResource struct {
	client *connect.Client
}

// The ID is "instance_id/hours_of_operation_id/hours_of_operation_override_id".
type ConnectHoursOfOperationOverrideResourceModel struct {
	ID                         frameworktypes.String       `tfsdk:"id"`
	HoursOfOperationOverrideID frameworktypes.String       `tfsdk:"hours_of_operation_override_id"`
	InstanceID                 frameworktypes.String       `tfsdk:"instance_id"`
	HoursOfOperationID         frameworktypes.String       `tfsdk:"hours_of_operation_id"`
	Name                       frameworktypes.String       `tfsdk:"name"`
	Description                frameworktypes.String       `tfsdk:"description"`
	EffectiveFrom              frameworktypes.String       `tfsdk:"effective_from"`
	EffectiveTill              frameworktypes.String       `tfsdk:"effective_till"`
	Config                     []OverrideOpeningHoursModel `tfsdk:"config"`
}

type OverrideOpeningHoursModel struct {
	Day       frameworktypes.String `tfsdk:"day"`
	StartTime OverrideTimeModel     `tfsdk:"start_time"`
	EndTime   OverrideTimeModel     `tfsdk:"end_time"`
}

type OverrideTimeModel struct {
	Hours   frameworktypes.Int32 `tfsdk:"hours"`
	Minutes frameworktypes.Int32 `tfsdk:"minutes"`
}

func (r *ConnectHoursOfOperationOverrideResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_hours_of_operation_override"
}

func (r *ConnectHoursOfOperationOverrideResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	timeAttributes := map[string]schema.Attribute{
		"hours": schema.Int32Attribute{
			MarkdownDescription: "The hour, 0 to 23.",
			Required:            true,
		},
		"minutes": schema.Int32Attribute{
			MarkdownDescription: "The minute, 0 to 59.",
			Required:            true,
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Replaces the weekly schedule of an hours of operation for a date range, such as a " +
			"Christmas closure or longer opening hours before a holiday.\n\n" +
			"During the range only the days in `config` are open. Without `config` the hours of operation are " +
			"closed for the whole range. A flow sees which override applied in `$.HoursOfOperation.AppliedOverride` " +
			"after a *Check hours of operation* block with *Check override* enabled.\n\n" +
			"The resource creates a one-off override (`STANDARD`). Recurring overrides are not supported.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The composite identifier (`instance_id/hours_of_operation_id/hours_of_operation_override_id`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hours_of_operation_override_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the override.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the Amazon Connect instance.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hours_of_operation_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the hours of operation the override belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the override, which a flow compares `$.HoursOfOperation.AppliedOverride` with.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the override.",
				Optional:            true,
			},
			"effective_from": schema.StringAttribute{
				MarkdownDescription: "The first day of the range, as `YYYY-MM-DD`.",
				Required:            true,
			},
			"effective_till": schema.StringAttribute{
				MarkdownDescription: "The last day of the range, as `YYYY-MM-DD`.",
				Required:            true,
			},
			"config": schema.SetNestedAttribute{
				MarkdownDescription: "The opening hours per day during the range. Omit it to close the whole range.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"day": schema.StringAttribute{
							MarkdownDescription: "The day of the week, e.g. `MONDAY`.",
							Required:            true,
						},
						"start_time": schema.SingleNestedAttribute{
							MarkdownDescription: "When the contact center opens on that day.",
							Required:            true,
							Attributes:          timeAttributes,
						},
						"end_time": schema.SingleNestedAttribute{
							MarkdownDescription: "When the contact center closes on that day.",
							Required:            true,
							Attributes:          timeAttributes,
						},
					},
				},
			},
		},
	}
}

func (r *ConnectHoursOfOperationOverrideResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = clients.Connect
}

func (r *ConnectHoursOfOperationOverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectHoursOfOperationOverrideResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateHoursOfOperationOverride(ctx, &connect.CreateHoursOfOperationOverrideInput{
		InstanceId:         data.InstanceID.ValueStringPointer(),
		HoursOfOperationId: data.HoursOfOperationID.ValueStringPointer(),
		Name:               data.Name.ValueStringPointer(),
		Description:        data.Description.ValueStringPointer(),
		EffectiveFrom:      data.EffectiveFrom.ValueStringPointer(),
		EffectiveTill:      data.EffectiveTill.ValueStringPointer(),
		Config:             overrideConfigToAPI(data.Config),
		OverrideType:       types.OverrideTypeStandard,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Hours of Operation Override", err.Error())
		return
	}

	data.HoursOfOperationOverrideID = frameworktypes.StringPointerValue(out.HoursOfOperationOverrideId)
	data.ID = frameworktypes.StringValue(composeHoursOfOperationOverrideID(
		data.InstanceID.ValueString(), data.HoursOfOperationID.ValueString(), data.HoursOfOperationOverrideID.ValueString()))

	tflog.Trace(ctx, "Created Hours of Operation Override", map[string]any{"id": data.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectHoursOfOperationOverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectHoursOfOperationOverrideResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.DescribeHoursOfOperationOverride(ctx, &connect.DescribeHoursOfOperationOverrideInput{
		InstanceId:                 data.InstanceID.ValueStringPointer(),
		HoursOfOperationId:         data.HoursOfOperationID.ValueStringPointer(),
		HoursOfOperationOverrideId: data.HoursOfOperationOverrideID.ValueStringPointer(),
	})
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Hours of Operation Override", err.Error())
		return
	}

	override := out.HoursOfOperationOverride
	data.Name = frameworktypes.StringPointerValue(override.Name)
	data.Description = frameworktypes.StringPointerValue(override.Description)
	data.EffectiveFrom = frameworktypes.StringPointerValue(override.EffectiveFrom)
	data.EffectiveTill = frameworktypes.StringPointerValue(override.EffectiveTill)
	data.Config = overrideConfigFromAPI(override.Config)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectHoursOfOperationOverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectHoursOfOperationOverrideResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateHoursOfOperationOverride(ctx, &connect.UpdateHoursOfOperationOverrideInput{
		InstanceId:                 data.InstanceID.ValueStringPointer(),
		HoursOfOperationId:         data.HoursOfOperationID.ValueStringPointer(),
		HoursOfOperationOverrideId: data.HoursOfOperationOverrideID.ValueStringPointer(),
		Name:                       data.Name.ValueStringPointer(),
		Description:                data.Description.ValueStringPointer(),
		EffectiveFrom:              data.EffectiveFrom.ValueStringPointer(),
		EffectiveTill:              data.EffectiveTill.ValueStringPointer(),
		Config:                     overrideConfigToAPI(data.Config),
		OverrideType:               types.OverrideTypeStandard,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Hours of Operation Override", err.Error())
		return
	}

	tflog.Trace(ctx, "Updated Hours of Operation Override", map[string]any{"id": data.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectHoursOfOperationOverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectHoursOfOperationOverrideResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteHoursOfOperationOverride(ctx, &connect.DeleteHoursOfOperationOverrideInput{
		InstanceId:                 data.InstanceID.ValueStringPointer(),
		HoursOfOperationId:         data.HoursOfOperationID.ValueStringPointer(),
		HoursOfOperationOverrideId: data.HoursOfOperationOverrideID.ValueStringPointer(),
	})
	if err != nil && !strings.Contains(err.Error(), "ResourceNotFoundException") {
		resp.Diagnostics.AddError("Error deleting Hours of Operation Override", err.Error())
		return
	}

	tflog.Trace(ctx, "Deleted Hours of Operation Override", map[string]any{"id": data.ID.ValueString()})
}

func (r *ConnectHoursOfOperationOverrideResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	instanceID, hoursOfOperationID, overrideID, err := parseHoursOfOperationOverrideID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), instanceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("hours_of_operation_id"), hoursOfOperationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("hours_of_operation_override_id"), overrideID)...)
}

func composeHoursOfOperationOverrideID(instanceID, hoursOfOperationID, overrideID string) string {
	return instanceID + "/" + hoursOfOperationID + "/" + overrideID
}

func parseHoursOfOperationOverrideID(id string) (instanceID, hoursOfOperationID, overrideID string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("expected instance_id/hours_of_operation_id/hours_of_operation_override_id, got: %q", id)
	}
	return parts[0], parts[1], parts[2], nil
}

// A closed range is an override without any opening hours, so the API gets an empty
// list rather than none at all, which it refuses.
func overrideConfigToAPI(config []OverrideOpeningHoursModel) []types.HoursOfOperationOverrideConfig {
	days := make([]types.HoursOfOperationOverrideConfig, 0, len(config))
	for _, day := range config {
		days = append(days, types.HoursOfOperationOverrideConfig{
			Day: types.OverrideDays(day.Day.ValueString()),
			StartTime: &types.OverrideTimeSlice{
				Hours:   day.StartTime.Hours.ValueInt32Pointer(),
				Minutes: day.StartTime.Minutes.ValueInt32Pointer(),
			},
			EndTime: &types.OverrideTimeSlice{
				Hours:   day.EndTime.Hours.ValueInt32Pointer(),
				Minutes: day.EndTime.Minutes.ValueInt32Pointer(),
			},
		})
	}
	return days
}

// A closed range comes back as an empty list, which stays null so that a config
// without `config` shows no difference.
func overrideConfigFromAPI(days []types.HoursOfOperationOverrideConfig) []OverrideOpeningHoursModel {
	if len(days) == 0 {
		return nil
	}
	config := make([]OverrideOpeningHoursModel, 0, len(days))
	for _, day := range days {
		config = append(config, OverrideOpeningHoursModel{
			Day:       frameworktypes.StringValue(string(day.Day)),
			StartTime: overrideTimeFromAPI(day.StartTime),
			EndTime:   overrideTimeFromAPI(day.EndTime),
		})
	}
	return config
}

func overrideTimeFromAPI(slice *types.OverrideTimeSlice) OverrideTimeModel {
	if slice == nil {
		return OverrideTimeModel{Hours: frameworktypes.Int32Value(0), Minutes: frameworktypes.Int32Value(0)}
	}
	return OverrideTimeModel{
		Hours:   frameworktypes.Int32Value(aws.ToInt32(slice.Hours)),
		Minutes: frameworktypes.Int32Value(aws.ToInt32(slice.Minutes)),
	}
}
