# Closed over Christmas: without config the whole range is closed.
resource "connectracer_connect_hours_of_operation_override" "christmas" {
  instance_id           = "12345678-1234-1234-1234-123456789012"
  hours_of_operation_id = aws_connect_hours_of_operation.support.hours_of_operation_id
  name                  = "Christmas"
  effective_from        = "2026-12-24"
  effective_till        = "2026-12-26"
}

# Shorter hours on New Year's Eve: only the listed days are open during the range.
resource "connectracer_connect_hours_of_operation_override" "new_years_eve" {
  instance_id           = "12345678-1234-1234-1234-123456789012"
  hours_of_operation_id = aws_connect_hours_of_operation.support.hours_of_operation_id
  name                  = "New Year's Eve"
  description           = "Open until noon"
  effective_from        = "2026-12-31"
  effective_till        = "2026-12-31"

  config = [
    {
      day        = "THURSDAY"
      start_time = { hours = 8, minutes = 0 }
      end_time   = { hours = 12, minutes = 0 }
    },
  ]
}
