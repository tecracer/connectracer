# AI Agent for Answer Recommendation
resource "connectracer_connect_ai_agent" "answer_recommendation" {
  assistant_id      = "12345678-1234-1234-1234-123456789012"
  name              = "answer-recommendation-agent"
  type              = "ANSWER_RECOMMENDATION"
  visibility_status = "PUBLISHED"
  description       = "AI Agent for answer recommendations"
  create_version    = true

  answer_recommendation_configuration {
    answer_generation_ai_prompt_id          = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
    intent_labeling_generation_ai_prompt_id = "ffffffff-1111-2222-3333-444444444444"
    query_reformulation_ai_prompt_id        = "55555555-6666-7777-8888-999999999999"
    locale                                  = "de_DE"

    association_configurations {
      association_id   = "abcdefab-1234-5678-abcd-123456789012"
      association_type = "KNOWLEDGE_BASE"

      knowledge_base_configuration {
        max_results                         = 5
        override_knowledge_base_search_type = "HYBRID"
      }
    }
  }

  tags = {
    Environment = "production"
  }
}

# AI Agent for Manual Search
resource "connectracer_connect_ai_agent" "manual_search" {
  assistant_id      = "12345678-1234-1234-1234-123456789012"
  name              = "manual-search-agent"
  type              = "MANUAL_SEARCH"
  visibility_status = "SAVED"
  description       = "AI Agent for manual search"

  manual_search_configuration {
    answer_generation_ai_prompt_id = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
    locale                         = "en_US"

    association_configurations {
      association_id   = "abcdefab-1234-5678-abcd-123456789012"
      association_type = "KNOWLEDGE_BASE"

      knowledge_base_configuration {
        max_results                         = 10
        override_knowledge_base_search_type = "SEMANTIC"
      }
    }
  }
}

# AI Agent for Self Service
resource "connectracer_connect_ai_agent" "self_service" {
  assistant_id      = "12345678-1234-1234-1234-123456789012"
  name              = "self-service-agent"
  type              = "SELF_SERVICE"
  visibility_status = "PUBLISHED"
  description       = "AI Agent for self-service"
  create_version    = true

  self_service_configuration {
    self_service_answer_generation_ai_prompt_id = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
    self_service_pre_processing_ai_prompt_id    = "ffffffff-1111-2222-3333-444444444444"

    association_configurations {
      association_id   = "abcdefab-1234-5678-abcd-123456789012"
      association_type = "KNOWLEDGE_BASE"

      knowledge_base_configuration {
        max_results                         = 5
        override_knowledge_base_search_type = "HYBRID"
      }
    }
  }
}
