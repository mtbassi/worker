## Automated WhatsApp Recovery System

### Overview

The Automated WhatsApp Recovery System is a WhatsApp-integrated solution designed to monitor customer progression across predefined journeys and automatically trigger recovery messages when abandonment is detected. The system enables progressive and personalized re-engagement strategies while preventing message duplication or spam through fine-grained rule control.

---

## Architecture

### Lambda 1: Event Tracker

**Responsibility**  
Manages the current state of a customer within a journey.

**Endpoints**

- **POST /journey/event**  
    Records the customer’s current step in a journey.
    
    **Payload (EventRequest):**
    
    `{   "journey_id": "onboarding-v2",   "step": "personal-data",   "customer_number": "5511999999999",   "tenant_id": "tenant-123",   "contact_id": "contact-456",   "last_interaction_at": "2025-01-15T10:30:00Z",   "metadata": {     "source": "whatsapp",     "campaign": "summer-2025"   } }`
    
    - Stores only the latest customer state in Redis, overwriting any previous state.
        
- **POST /journey/finish**  
    Marks a journey as completed.
    
    **Payload (FinishRequest):**
    
    `{   "journey_id": "onboarding-v2",   "customer_number": "5511999999999" }`
    
    - Removes the customer state from Redis to prevent further recovery messages.
        

---

### Redis Data Structures

#### Current State

- **Key Pattern:**  
    `journey:{journey_id}:{customer_number}:state`
    
- **Behavior:**
    
    - Holds the serialized `EventRequest`
        
    - Configurable TTL per journey
        
    - Always represents the latest known customer state
        

#### Recovery History

- **Key Pattern:**  
    `journey:{journey_id}:{customer_number}:repiques`
    
- **Type:**  
    Append-only list
    
- **Purpose:**
    
    - Track recovery attempts and execution history
        
    - Support validation of timing, limits, and sequencing rules
        

---

## Recovery Rules Model (Per-Step, Multi-Rule)

Each journey step can define **multiple named recovery rules**, each with its own inactivity window, execution limits, and associated message template. This model allows different recovery strategies to be applied progressively over time within the same step.

For example, a single step may define:

- A **10-minute rule**, executed only once
    
- A **30-minute rule**, executed later if inactivity persists
    

Because each rule is evaluated independently, the 10-minute rule will never execute concurrently with the 30-minute rule. Each rule has its own timing, attempt counter, and execution history.

### Named Rules and Template Association

Recovery rules are explicitly named and mapped to specific templates. This enables multiple rules within the same step to trigger different messages, such as:

- Lighter, shorter messages in early recovery attempts
    
- More direct messages or alternative CTAs in later attempts
    

Each rule maintains an isolated execution context, ensuring full separation between recovery strategies within the same step.

### Model Implications

This approach enables:

- Execution control at the **rule level**, not just at the step level
    
- Independent recovery history per rule
    
- Explicit **rule → template** association
    
- Progressive re-engagement strategies without overlap
    
- Guaranteed prevention of simultaneous or duplicated messages
    

---

## Lambda 2: Recovery Message Sender

**Responsibility**  
Identifies abandoned journeys and sends personalized recovery messages.

**Execution Flow (every 5 minutes)**

1. **Redis Scan**
    
    - Scans keys matching `journey:*:*:state` using `SCAN` with pipelining.
        
    - Loads the current `EventRequest` for each active customer.
        
2. **Rule Evaluation**  
    For each active customer:
    
    - Loads journey, step, and rule definitions from AWS AppConfig.
        
    - Calculates inactivity based on `last_interaction_at`.
        
    - Queries recovery history to evaluate rule-specific execution state.
        
    
    **Validation Conditions**
    
    - Rule-specific inactivity threshold reached
        
    - Maximum attempts for the rule not exceeded
        
    - Minimum interval between executions respected
        
3. **Message Dispatch**
    
    - Selects the template associated with the matched rule.
        
    - Renders the template using metadata.
        
    - Sends the message via the WhatsApp Business API.
        
    - Records execution in the rule-specific recovery history.
        
    - Optionally updates `last_interaction_at` to prevent flooding.
        

---

## Configuration via AWS AppConfig

### Rules Configuration

**File:** `journey.<journey-name>.yaml`

`journey: onboarding-v2 global:   max_total_attempts: 5   min_interval_between_attempts_minutes: 15 steps:   - name: personal-data     recovery_rules:       - name: early-reminder         inactive_minutes: 10         max_attempts: 1         template: personal-data-soft       - name: late-reminder         inactive_minutes: 30         max_attempts: 2         template: personal-data-cta   - name: document-upload     recovery_rules:       - name: first-reminder         inactive_minutes: 60         max_attempts: 1         template: document-upload-soft`

### Templates Configuration

**File:** `journey.<journey-name>.templates.yaml`

`journey: onboarding-v2 templates:   personal-data-soft: |     Hello! We noticed you didn’t complete your registration. Can we help with something?   personal-data-cta: |     Complete your registration now to continue. Click here: {{.metadata.link}}   document-upload-soft: |     We’re almost there. Just upload your documents to proceed.`

---

## Technical Components

- **Language:** Go (both Lambdas)
    
- **Cache:** Redis (state and recovery history)
    
- **Configuration:** AWS AppConfig with hot reload
    
- **Messaging:** WhatsApp Business API
    
- **Scheduler:** Amazon EventBridge (5-minute interval)
    

---

## Core Data Structures (Go)

``type EventRequest struct {     JourneyID         string                 `json:"journey_id"`     Step              string                 `json:"step"`     CustomerNumber    string                 `json:"customer_number"`     TenantID          string                 `json:"tenant_id"`     ContactID         string                 `json:"contact_id"`     LastInteractionAt time.Time              `json:"last_interaction_at"`     Metadata          map[string]interface{} `json:"metadata"` }  type FinishRequest struct {     JourneyID      string `json:"journey_id"`     CustomerNumber string `json:"customer_number"` }  type RepiqueEntry struct {     Step          string    `json:"step"`     Rule          string    `json:"rule"`     SentAt        time.Time `json:"sent_at"`     TemplateUsed  string    `json:"template_used"`     AttemptNumber int       `json:"attempt_number"` }``

---

## Benefits

- Automated customer recovery via WhatsApp
    
- Rule-level execution control and isolation
    
- Progressive re-engagement strategies per step
    
- Full recovery history per rule for auditing and analytics
    
- Flexible, hot-reloadable journey configuration
    
- Low-latency processing with Redis
    
- Fully serverless and horizontally scalable architecture
    
- Dynamic templates with metadata-driven personalization