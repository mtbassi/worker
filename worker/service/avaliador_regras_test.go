package service

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"worker-project/shared/domain"
	"worker-project/worker/config"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestEvaluateRecoveryRule_DisabledRule(t *testing.T) {
	rule := &config.RecoveryRule{
		Name:            "test-rule",
		Enabled:         false,
		InactiveMinutes: 10,
		MaxAttempts:     3,
		Template:        "test-template",
	}
	globalCfg := &config.GlobalConfig{
		Enabled:                           true,
		MaxTotalAttempts:                  5,
		MinIntervalBetweenAttemptsMinutes: 5,
	}
	state := &domain.JourneyState{
		LastInteractionAt: time.Now().Add(-15 * time.Minute),
	}
	history := &domain.RepiqueHistory{Entries: []domain.RepiqueEntry{}}

	result := EvaluateRecoveryRule(rule, globalCfg, state, history)

	if result.ShouldTrigger {
		t.Error("regra desabilitada não deve disparar")
	}
	if result.Reason != "regra desabilitada" {
		t.Errorf("esperado reason 'regra desabilitada', obteve '%s'", result.Reason)
	}
}

func TestEvaluateRecoveryRule_GlobalMaxAttemptsExceeded(t *testing.T) {
	rule := &config.RecoveryRule{
		Name:            "test-rule",
		Enabled:         true,
		InactiveMinutes: 10,
		MaxAttempts:     3,
		Template:        "test-template",
	}
	globalCfg := &config.GlobalConfig{
		Enabled:                           true,
		MaxTotalAttempts:                  2,
		MinIntervalBetweenAttemptsMinutes: 5,
	}
	state := &domain.JourneyState{
		LastInteractionAt: time.Now().Add(-15 * time.Minute),
	}
	history := &domain.RepiqueHistory{
		Entries: []domain.RepiqueEntry{
			{Rule: "rule1", SentAt: time.Now().Add(-1 * time.Hour)},
			{Rule: "rule2", SentAt: time.Now().Add(-30 * time.Minute)},
		},
	}

	result := EvaluateRecoveryRule(rule, globalCfg, state, history)

	if result.ShouldTrigger {
		t.Error("regra não deve disparar quando máximo global excedido")
	}
	if result.Reason != "máximo global de tentativas excedido" {
		t.Errorf("esperado reason 'máximo global de tentativas excedido', obteve '%s'", result.Reason)
	}
}

func TestEvaluateRecoveryRule_RuleMaxAttemptsExceeded(t *testing.T) {
	rule := &config.RecoveryRule{
		Name:            "test-rule",
		Enabled:         true,
		InactiveMinutes: 10,
		MaxAttempts:     2,
		Template:        "test-template",
	}
	globalCfg := &config.GlobalConfig{
		Enabled:                           true,
		MaxTotalAttempts:                  10,
		MinIntervalBetweenAttemptsMinutes: 5,
	}
	state := &domain.JourneyState{
		LastInteractionAt: time.Now().Add(-15 * time.Minute),
	}
	history := &domain.RepiqueHistory{
		Entries: []domain.RepiqueEntry{
			{Rule: "test-rule", SentAt: time.Now().Add(-1 * time.Hour)},
			{Rule: "test-rule", SentAt: time.Now().Add(-30 * time.Minute)},
		},
	}

	result := EvaluateRecoveryRule(rule, globalCfg, state, history)

	if result.ShouldTrigger {
		t.Error("regra não deve disparar quando máximo de tentativas da regra excedido")
	}
	if result.Reason != "máximo de tentativas da regra excedido" {
		t.Errorf("esperado reason 'máximo de tentativas da regra excedido', obteve '%s'", result.Reason)
	}
}

func TestEvaluateRecoveryRule_MinIntervalNotReached(t *testing.T) {
	rule := &config.RecoveryRule{
		Name:            "test-rule",
		Enabled:         true,
		InactiveMinutes: 10,
		MaxAttempts:     5,
		Template:        "test-template",
	}
	globalCfg := &config.GlobalConfig{
		Enabled:                           true,
		MaxTotalAttempts:                  10,
		MinIntervalBetweenAttemptsMinutes: 15,
	}
	state := &domain.JourneyState{
		LastInteractionAt: time.Now().Add(-20 * time.Minute),
	}
	// Última tentativa foi 5 minutos atrás, mas intervalo mínimo é 15 minutos
	history := &domain.RepiqueHistory{
		Entries: []domain.RepiqueEntry{
			{Rule: "test-rule", SentAt: time.Now().Add(-5 * time.Minute)},
		},
	}

	result := EvaluateRecoveryRule(rule, globalCfg, state, history)

	if result.ShouldTrigger {
		t.Error("regra não deve disparar quando intervalo mínimo não atingido")
	}
	if result.Reason != "intervalo mínimo não atingido" {
		t.Errorf("esperado reason 'intervalo mínimo não atingido', obteve '%s'", result.Reason)
	}
}

func TestEvaluateRecoveryRule_InactivityThresholdNotReached(t *testing.T) {
	rule := &config.RecoveryRule{
		Name:            "test-rule",
		Enabled:         true,
		InactiveMinutes: 30,
		MaxAttempts:     5,
		Template:        "test-template",
	}
	globalCfg := &config.GlobalConfig{
		Enabled:                           true,
		MaxTotalAttempts:                  10,
		MinIntervalBetweenAttemptsMinutes: 5,
	}
	// Última interação foi 15 minutos atrás, mas regra requer 30 minutos
	state := &domain.JourneyState{
		LastInteractionAt: time.Now().Add(-15 * time.Minute),
	}
	history := &domain.RepiqueHistory{Entries: []domain.RepiqueEntry{}}

	result := EvaluateRecoveryRule(rule, globalCfg, state, history)

	if result.ShouldTrigger {
		t.Error("regra não deve disparar quando tempo de inatividade não atingido")
	}
	if result.Reason != "tempo de inatividade não atingido" {
		t.Errorf("esperado reason 'tempo de inatividade não atingido', obteve '%s'", result.Reason)
	}
}

func TestEvaluateRecoveryRule_AllConditionsMet(t *testing.T) {
	rule := &config.RecoveryRule{
		Name:            "test-rule",
		Enabled:         true,
		InactiveMinutes: 10,
		MaxAttempts:     5,
		Template:        "test-template",
	}
	globalCfg := &config.GlobalConfig{
		Enabled:                           true,
		MaxTotalAttempts:                  10,
		MinIntervalBetweenAttemptsMinutes: 5,
	}
	state := &domain.JourneyState{
		LastInteractionAt: time.Now().Add(-15 * time.Minute),
	}
	history := &domain.RepiqueHistory{Entries: []domain.RepiqueEntry{}}

	result := EvaluateRecoveryRule(rule, globalCfg, state, history)

	if !result.ShouldTrigger {
		t.Errorf("regra deve disparar quando todas condições atendidas, reason: %s", result.Reason)
	}
	if result.Reason != "todas as condições atendidas" {
		t.Errorf("esperado reason 'todas as condições atendidas', obteve '%s'", result.Reason)
	}
}

func TestFindTriggeredRule_NoRulesTriggered(t *testing.T) {
	rules := []config.RecoveryRule{
		{Name: "rule1", Enabled: false, InactiveMinutes: 10, MaxAttempts: 3},
		{Name: "rule2", Enabled: false, InactiveMinutes: 20, MaxAttempts: 3},
	}
	globalCfg := &config.GlobalConfig{
		Enabled:          true,
		MaxTotalAttempts: 10,
	}
	state := &domain.JourneyState{
		LastInteractionAt: time.Now().Add(-30 * time.Minute),
	}
	history := &domain.RepiqueHistory{Entries: []domain.RepiqueEntry{}}
	logger := newTestLogger()

	result := FindTriggeredRule(rules, globalCfg, state, history, logger)

	if result != nil {
		t.Error("esperado nil quando nenhuma regra dispara")
	}
}

func TestFindTriggeredRule_SingleRuleTriggered(t *testing.T) {
	rules := []config.RecoveryRule{
		{Name: "rule1", Enabled: true, InactiveMinutes: 10, MaxAttempts: 3, Template: "t1"},
		{Name: "rule2", Enabled: false, InactiveMinutes: 20, MaxAttempts: 3, Template: "t2"},
	}
	globalCfg := &config.GlobalConfig{
		Enabled:          true,
		MaxTotalAttempts: 10,
	}
	state := &domain.JourneyState{
		LastInteractionAt: time.Now().Add(-15 * time.Minute),
	}
	history := &domain.RepiqueHistory{Entries: []domain.RepiqueEntry{}}
	logger := newTestLogger()

	result := FindTriggeredRule(rules, globalCfg, state, history, logger)

	if result == nil {
		t.Fatal("esperado resultado quando uma regra dispara")
	}
	if result.Rule.Name != "rule1" {
		t.Errorf("esperado rule1, obteve %s", result.Rule.Name)
	}
}

func TestFindTriggeredRule_MultipleRulesTriggered_SelectsHighestInactivity(t *testing.T) {
	rules := []config.RecoveryRule{
		{Name: "rule-10min", Enabled: true, InactiveMinutes: 10, MaxAttempts: 3, Template: "t1"},
		{Name: "rule-30min", Enabled: true, InactiveMinutes: 30, MaxAttempts: 3, Template: "t2"},
		{Name: "rule-20min", Enabled: true, InactiveMinutes: 20, MaxAttempts: 3, Template: "t3"},
	}
	globalCfg := &config.GlobalConfig{
		Enabled:          true,
		MaxTotalAttempts: 10,
	}
	// 35 minutos de inatividade - todas as regras são elegíveis
	state := &domain.JourneyState{
		JourneyID:         "test-journey",
		CustomerNumber:    "5511999999999",
		Step:              "test-step",
		LastInteractionAt: time.Now().Add(-35 * time.Minute),
	}
	history := &domain.RepiqueHistory{Entries: []domain.RepiqueEntry{}}
	logger := newTestLogger()

	result := FindTriggeredRule(rules, globalCfg, state, history, logger)

	if result == nil {
		t.Fatal("esperado resultado quando múltiplas regras disparam")
	}
	// Deve selecionar a regra com maior InactiveMinutes (30min)
	if result.Rule.Name != "rule-30min" {
		t.Errorf("esperado rule-30min (maior inatividade), obteve %s", result.Rule.Name)
	}
	if result.Rule.InactiveMinutes != 30 {
		t.Errorf("esperado InactiveMinutes=30, obteve %d", result.Rule.InactiveMinutes)
	}
}

func TestFindTriggeredRule_MultipleRulesTriggered_PartialEligibility(t *testing.T) {
	rules := []config.RecoveryRule{
		{Name: "rule-10min", Enabled: true, InactiveMinutes: 10, MaxAttempts: 3, Template: "t1"},
		{Name: "rule-30min", Enabled: true, InactiveMinutes: 30, MaxAttempts: 3, Template: "t2"},
		{Name: "rule-60min", Enabled: true, InactiveMinutes: 60, MaxAttempts: 3, Template: "t3"},
	}
	globalCfg := &config.GlobalConfig{
		Enabled:          true,
		MaxTotalAttempts: 10,
	}
	// 35 minutos de inatividade - apenas rule-10min e rule-30min são elegíveis
	state := &domain.JourneyState{
		JourneyID:         "test-journey",
		CustomerNumber:    "5511999999999",
		Step:              "test-step",
		LastInteractionAt: time.Now().Add(-35 * time.Minute),
	}
	history := &domain.RepiqueHistory{Entries: []domain.RepiqueEntry{}}
	logger := newTestLogger()

	result := FindTriggeredRule(rules, globalCfg, state, history, logger)

	if result == nil {
		t.Fatal("esperado resultado quando regras disparam")
	}
	// Deve selecionar rule-30min (maior entre as elegíveis)
	if result.Rule.Name != "rule-30min" {
		t.Errorf("esperado rule-30min, obteve %s", result.Rule.Name)
	}
}
