// Copyright tecRacer Group 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ConnectDataTableRecordResource{}
var _ resource.ResourceWithImportState = &ConnectDataTableRecordResource{}

func NewConnectDataTableRecordResource() resource.Resource {
	return &ConnectDataTableRecordResource{}
}

type ConnectDataTableRecordResource struct {
	client *connect.Client
}

// The ID is "instance_id/data_table_id/record_id".
type ConnectDataTableRecordResourceModel struct {
	ID            frameworktypes.String            `tfsdk:"id"`
	RecordID      frameworktypes.String            `tfsdk:"record_id"`
	InstanceID    frameworktypes.String            `tfsdk:"instance_id"`
	DataTableID   frameworktypes.String            `tfsdk:"data_table_id"`
	PrimaryValues map[string]frameworktypes.String `tfsdk:"primary_values"`
	Values        map[string]frameworktypes.String `tfsdk:"values"`
}

func (r *ConnectDataTableRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_data_table_record"
}

func (r *ConnectDataTableRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A record of an Amazon Connect data table: its primary values and the values of its other attributes.\n\n" +
			"Attributes are addressed by name. Every write sends the lock version Connect currently holds for each " +
			"value, read right before the write, and writes all values of the record in one call. A value changed " +
			"outside Terraform is therefore overwritten on the next apply rather than failing it. To leave values to the " +
			"admin website after the create, set `lifecycle { ignore_changes = [values] }`.\n\n" +
			"A record whose primary values exist already is taken over on create rather than refused. A write that " +
			"conflicts with another one, e.g. from a parallel apply, is retried with the record read again.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The composite identifier (`instance_id/data_table_id/record_id`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"record_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the record.",
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
			"data_table_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the data table.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"primary_values": schema.MapAttribute{
				MarkdownDescription: "The value of every primary attribute, keyed by attribute name. Other primary values make another record, so a change replaces it.",
				Required:            true,
				ElementType:         frameworktypes.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
			},
			"values": schema.MapAttribute{
				MarkdownDescription: "The values of the other attributes, keyed by attribute name, as text whatever the attribute's type (`\"true\"` for a boolean). Attributes left out are not touched.",
				Required:            true,
				ElementType:         frameworktypes.StringType,
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
			},
		},
	}
}

func (r *ConnectDataTableRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConnectDataTableRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectDataTableRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	find := func() ([]types.DataTableValueSummary, error) { return r.findRecord(ctx, data) }
	if err := r.syncValues(ctx, data, nil, find); err != nil {
		resp.Diagnostics.AddError("Error creating Data Table Record", err.Error())
		return
	}

	record, err := find()
	if err != nil {
		resp.Diagnostics.AddError("Error creating Data Table Record", err.Error())
		return
	}
	if len(record) == 0 || record[0].RecordId == nil {
		resp.Diagnostics.AddError("Error creating Data Table Record", "Connect reported no record identifier.")
		return
	}

	data.RecordID = frameworktypes.StringPointerValue(record[0].RecordId)
	data.ID = frameworktypes.StringValue(composeDataTableRecordID(
		data.InstanceID.ValueString(), data.DataTableID.ValueString(), data.RecordID.ValueString()))

	tflog.Trace(ctx, "Created Data Table Record", map[string]any{"id": data.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectDataTableRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectDataTableRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	record, err := r.readRecord(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Data Table Record", err.Error())
		return
	}
	if len(record) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	data.PrimaryValues = dataTablePrimaryValuesFromAPI(record)
	data.Values = dataTableValuesFromAPI(record, data.Values)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectDataTableRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ConnectDataTableRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	read := func() ([]types.DataTableValueSummary, error) { return r.readRecord(ctx, state) }
	if err := r.syncValues(ctx, plan, state.Values, read); err != nil {
		resp.Diagnostics.AddError("Error updating Data Table Record", err.Error())
		return
	}

	plan.ID = state.ID
	plan.RecordID = state.RecordID

	tflog.Trace(ctx, "Updated Data Table Record", map[string]any{"id": plan.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConnectDataTableRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectDataTableRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := retryDataTableConflicts(ctx, func() error {
		record, err := r.readRecord(ctx, data)
		if err != nil || len(record) == 0 {
			return err
		}

		locks := dataTableLockVersions(record)
		names := make([]string, 0, len(locks))
		for name := range locks {
			names = append(names, name)
		}

		out, err := r.client.BatchDeleteDataTableValue(ctx, &connect.BatchDeleteDataTableValueInput{
			InstanceId:  data.InstanceID.ValueStringPointer(),
			DataTableId: data.DataTableID.ValueStringPointer(),
			Values:      dataTableDeleteIdentifiers(names, dataTablePrimaryValuesToAPI(dataTablePrimaryValuesFromAPI(record)), locks),
		})
		if err != nil {
			if strings.Contains(err.Error(), "ResourceNotFoundException") {
				return nil
			}
			return err
		}
		if failed := dataTableFailures(out.Failed); failed != "" {
			return errors.New(failed)
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Data Table Record", err.Error())
		return
	}

	tflog.Trace(ctx, "Deleted Data Table Record", map[string]any{"id": data.ID.ValueString()})
}

func (r *ConnectDataTableRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	instanceID, dataTableID, recordID, err := parseDataTableRecordID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), instanceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("data_table_id"), dataTableID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("record_id"), recordID)...)
}

// Connect refuses concurrent writes to the same attribute, even across records.
const (
	dataTableWriteAttempts = 8
	dataTableFirstBackoff  = 500 * time.Millisecond
	dataTableMaxBackoff    = 8 * time.Second
)

func (r *ConnectDataTableRecordResource) syncValues(ctx context.Context, plan ConnectDataTableRecordResourceModel, previous map[string]frameworktypes.String, fetch func() ([]types.DataTableValueSummary, error)) error {
	return retryDataTableConflicts(ctx, func() error {
		record, err := fetch()
		if err != nil {
			return err
		}
		return r.writeValues(ctx, plan, previous, record)
	})
}

func retryDataTableConflicts(ctx context.Context, write func() error) error {
	backoff := dataTableFirstBackoff
	for attempt := 1; ; attempt++ {
		err := write()
		if err == nil || !isDataTableConflict(err) || attempt == dataTableWriteAttempts {
			return err
		}

		wait := backoff/2 + time.Duration(rand.Int64N(int64(backoff)))
		tflog.Debug(ctx, "Data table write conflicted, retrying", map[string]any{"attempt": attempt, "wait": wait.String(), "error": err.Error()})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		backoff = min(backoff*2, dataTableMaxBackoff)
	}
}

func isDataTableConflict(err error) bool {
	message := err.Error()
	return strings.Contains(message, "Concurrency conflict") || strings.Contains(message, "lock does not match")
}

func (r *ConnectDataTableRecordResource) writeValues(ctx context.Context, plan ConnectDataTableRecordResourceModel, previous map[string]frameworktypes.String, record []types.DataTableValueSummary) error {
	locks := dataTableLockVersions(record)
	primaryValues := dataTablePrimaryValuesToAPI(plan.PrimaryValues)
	changes := diffDataTableValues(previous, plan.Values, record)

	if len(changes.update) > 0 {
		out, err := r.client.BatchUpdateDataTableValue(ctx, &connect.BatchUpdateDataTableValueInput{
			InstanceId:  plan.InstanceID.ValueStringPointer(),
			DataTableId: plan.DataTableID.ValueStringPointer(),
			Values:      dataTableValuesToAPI(changes.update, primaryValues, locks),
		})
		if err != nil {
			return err
		}
		if failed := dataTableFailures(out.Failed); failed != "" {
			return errors.New(failed)
		}
	}

	if len(changes.create) > 0 {
		out, err := r.client.BatchCreateDataTableValue(ctx, &connect.BatchCreateDataTableValueInput{
			InstanceId:  plan.InstanceID.ValueStringPointer(),
			DataTableId: plan.DataTableID.ValueStringPointer(),
			Values:      dataTableValuesToAPI(changes.create, primaryValues, nil),
		})
		if err != nil {
			return err
		}
		if failed := dataTableFailures(out.Failed); failed != "" {
			return errors.New(failed)
		}
	}

	if len(changes.remove) > 0 {
		out, err := r.client.BatchDeleteDataTableValue(ctx, &connect.BatchDeleteDataTableValueInput{
			InstanceId:  plan.InstanceID.ValueStringPointer(),
			DataTableId: plan.DataTableID.ValueStringPointer(),
			Values:      dataTableDeleteIdentifiers(changes.remove, primaryValues, locks),
		})
		if err != nil {
			return err
		}
		if failed := dataTableFailures(out.Failed); failed != "" {
			return errors.New(failed)
		}
	}

	return nil
}

func (r *ConnectDataTableRecordResource) findRecord(ctx context.Context, data ConnectDataTableRecordResourceModel) ([]types.DataTableValueSummary, error) {
	filters := make([]types.PrimaryAttributeValueFilter, 0, len(data.PrimaryValues))
	for _, name := range sortedKeys(data.PrimaryValues) {
		filters = append(filters, types.PrimaryAttributeValueFilter{
			AttributeName: aws.String(name),
			Values:        []string{data.PrimaryValues[name].ValueString()},
		})
	}
	return r.listValues(ctx, &connect.ListDataTableValuesInput{
		InstanceId:             data.InstanceID.ValueStringPointer(),
		DataTableId:            data.DataTableID.ValueStringPointer(),
		PrimaryAttributeValues: filters,
	})
}

func (r *ConnectDataTableRecordResource) readRecord(ctx context.Context, data ConnectDataTableRecordResourceModel) ([]types.DataTableValueSummary, error) {
	return r.listValues(ctx, &connect.ListDataTableValuesInput{
		InstanceId:  data.InstanceID.ValueStringPointer(),
		DataTableId: data.DataTableID.ValueStringPointer(),
		RecordIds:   []string{data.RecordID.ValueString()},
	})
}

func (r *ConnectDataTableRecordResource) listValues(ctx context.Context, input *connect.ListDataTableValuesInput) ([]types.DataTableValueSummary, error) {
	var values []types.DataTableValueSummary
	paginator := connect.NewListDataTableValuesPaginator(r.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "ResourceNotFoundException") {
				return nil, nil
			}
			return nil, err
		}
		values = append(values, page.Values...)
	}
	return values, nil
}

func composeDataTableRecordID(instanceID, dataTableID, recordID string) string {
	return instanceID + "/" + dataTableID + "/" + recordID
}

func parseDataTableRecordID(id string) (instanceID, dataTableID, recordID string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("expected instance_id/data_table_id/record_id, got: %q", id)
	}
	return parts[0], parts[1], parts[2], nil
}

func dataTablePrimaryValuesToAPI(primaryValues map[string]frameworktypes.String) []types.PrimaryValue {
	names := sortedKeys(primaryValues)
	out := make([]types.PrimaryValue, 0, len(names))
	for _, name := range names {
		out = append(out, types.PrimaryValue{
			AttributeName: aws.String(name),
			Value:         primaryValues[name].ValueStringPointer(),
		})
	}
	return out
}

func dataTableValuesToAPI(values map[string]frameworktypes.String, primaryValues []types.PrimaryValue, locks map[string]*types.DataTableLockVersion) []types.DataTableValue {
	names := sortedKeys(values)
	out := make([]types.DataTableValue, 0, len(names))
	for _, name := range names {
		out = append(out, types.DataTableValue{
			AttributeName: aws.String(name),
			Value:         values[name].ValueStringPointer(),
			PrimaryValues: primaryValues,
			LockVersion:   locks[name],
		})
	}
	return out
}

func dataTableDeleteIdentifiers(names []string, primaryValues []types.PrimaryValue, locks map[string]*types.DataTableLockVersion) []types.DataTableDeleteValueIdentifier {
	sort.Strings(names)
	out := make([]types.DataTableDeleteValueIdentifier, 0, len(names))
	for _, name := range names {
		out = append(out, types.DataTableDeleteValueIdentifier{
			AttributeName: aws.String(name),
			PrimaryValues: primaryValues,
			LockVersion:   locks[name],
		})
	}
	return out
}

func dataTablePrimaryValuesFromAPI(record []types.DataTableValueSummary) map[string]frameworktypes.String {
	out := map[string]frameworktypes.String{}
	if len(record) == 0 {
		return out
	}
	for _, primary := range record[0].PrimaryValues {
		out[aws.ToString(primary.AttributeName)] = frameworktypes.StringPointerValue(primary.Value)
	}
	return out
}

// Only managed attributes, or every one after an import.
func dataTableValuesFromAPI(record []types.DataTableValueSummary, managed map[string]frameworktypes.String) map[string]frameworktypes.String {
	out := map[string]frameworktypes.String{}
	for _, value := range record {
		name := aws.ToString(value.AttributeName)
		if len(managed) > 0 {
			if _, ok := managed[name]; !ok {
				continue
			}
		}
		out[name] = frameworktypes.StringPointerValue(value.Value)
	}
	return out
}

func dataTableLockVersions(record []types.DataTableValueSummary) map[string]*types.DataTableLockVersion {
	out := make(map[string]*types.DataTableLockVersion, len(record))
	for _, value := range record {
		out[aws.ToString(value.AttributeName)] = value.LockVersion
	}
	return out
}

type dataTableValueChanges struct {
	update map[string]frameworktypes.String
	create map[string]frameworktypes.String
	remove []string
}

// Against what Connect holds, so a matching value is not written and cannot conflict.
func diffDataTableValues(previous, plan map[string]frameworktypes.String, record []types.DataTableValueSummary) dataTableValueChanges {
	current := make(map[string]*string, len(record))
	for _, value := range record {
		current[aws.ToString(value.AttributeName)] = value.Value
	}

	changes := dataTableValueChanges{
		update: map[string]frameworktypes.String{},
		create: map[string]frameworktypes.String{},
	}
	for name, value := range plan {
		held, ok := current[name]
		switch {
		case !ok:
			changes.create[name] = value
		case aws.ToString(held) != value.ValueString():
			changes.update[name] = value
		}
	}
	for name := range previous {
		if _, kept := plan[name]; kept {
			continue
		}
		if _, held := current[name]; held {
			changes.remove = append(changes.remove, name)
		}
	}
	sort.Strings(changes.remove)
	return changes
}

type dataTableFailure interface {
	types.BatchCreateDataTableValueFailureResult | types.BatchUpdateDataTableValueFailureResult | types.BatchDeleteDataTableValueFailureResult
}

// A batch call succeeds even when single values fail.
func dataTableFailures[F dataTableFailure](failed []F) string {
	messages := make([]string, 0, len(failed))
	for _, f := range failed {
		switch v := any(f).(type) {
		case types.BatchCreateDataTableValueFailureResult:
			messages = append(messages, fmt.Sprintf("%s: %s", aws.ToString(v.AttributeName), aws.ToString(v.Message)))
		case types.BatchUpdateDataTableValueFailureResult:
			messages = append(messages, fmt.Sprintf("%s: %s", aws.ToString(v.AttributeName), aws.ToString(v.Message)))
		case types.BatchDeleteDataTableValueFailureResult:
			messages = append(messages, fmt.Sprintf("%s: %s", aws.ToString(v.AttributeName), aws.ToString(v.Message)))
		}
	}
	return strings.Join(messages, "; ")
}

func sortedKeys(values map[string]frameworktypes.String) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
