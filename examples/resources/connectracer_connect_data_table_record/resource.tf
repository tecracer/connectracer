# The closed message of a support line: a table keyed by hours and override, with an
# active flag and the text a flow plays while closed.
resource "connectracer_connect_data_table_record" "support_closed" {
  instance_id   = "12345678-1234-1234-1234-123456789012"
  data_table_id = "abcdef12-3456-7890-abcd-ef1234567890"

  primary_values = {
    hours    = "support"
    override = "*"
  }

  values = {
    active  = "true"
    message = "We are closed right now. Please call again during our opening hours."
  }
}
