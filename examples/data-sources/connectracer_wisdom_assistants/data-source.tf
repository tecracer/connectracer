data "connectracer_wisdom_assistants" "example" {
}

output "assistants" {
  value = data.connectracer_wisdom_assistants.example.assistants
}

output "assistant_names" {
  value = [for assistant in data.connectracer_wisdom_assistants.example.assistants : assistant.name]
}
