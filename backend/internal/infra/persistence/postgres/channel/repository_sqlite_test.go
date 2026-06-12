package channel

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListModelsSQLiteUsesPortableRouteStats(t *testing.T) {
	db := openChannelSQLiteTestDB(t)
	ctx := context.Background()

	activeUpstream := model.LLMUpstream{Name: "active-upstream", Status: "active"}
	inactiveUpstream := model.LLMUpstream{Name: "inactive-upstream", Status: "inactive"}
	if err := db.Create(&activeUpstream).Error; err != nil {
		t.Fatalf("create active upstream: %v", err)
	}
	if err := db.Create(&inactiveUpstream).Error; err != nil {
		t.Fatalf("create inactive upstream: %v", err)
	}

	upstreamModels := []model.LLMUpstreamModel{
		{UpstreamID: activeUpstream.ID, BindingCode: "active-a", UpstreamModelName: "active-a", Status: "active"},
		{UpstreamID: activeUpstream.ID, BindingCode: "active-b", UpstreamModelName: "active-b", Status: "active"},
		{UpstreamID: activeUpstream.ID, BindingCode: "inactive-model", UpstreamModelName: "inactive-model", Status: "inactive"},
		{UpstreamID: inactiveUpstream.ID, BindingCode: "inactive-upstream-model", UpstreamModelName: "inactive-upstream-model", Status: "active"},
	}
	if err := db.Create(&upstreamModels).Error; err != nil {
		t.Fatalf("create upstream models: %v", err)
	}
	activeModelA := upstreamModels[0]
	activeModelB := upstreamModels[1]
	inactiveModel := upstreamModels[2]
	inactiveUpstreamModel := upstreamModels[3]

	platformModel := model.LLMPlatformModel{Name: "gpt-test", Vendor: "openai", Status: "active", SortOrder: 1}
	emptyPlatformModel := model.LLMPlatformModel{Name: "empty-test", Vendor: "openai", Status: "active", SortOrder: 2}
	if err := db.Create(&platformModel).Error; err != nil {
		t.Fatalf("create platform model: %v", err)
	}
	if err := db.Create(&emptyPlatformModel).Error; err != nil {
		t.Fatalf("create empty platform model: %v", err)
	}

	routes := []model.LLMPlatformModelRoute{
		{PlatformModelID: platformModel.ID, UpstreamModelID: activeModelA.ID, Protocol: "openai_responses", Status: "active"},
		{PlatformModelID: platformModel.ID, UpstreamModelID: activeModelB.ID, Protocol: "openai_responses", Status: "active"},
		{PlatformModelID: platformModel.ID, UpstreamModelID: activeModelA.ID, Protocol: "xai_responses", Status: "active"},
		{PlatformModelID: platformModel.ID, UpstreamModelID: inactiveModel.ID, Protocol: "anthropic_messages", Status: "active"},
		{PlatformModelID: platformModel.ID, UpstreamModelID: inactiveUpstreamModel.ID, Protocol: "google_generate_content", Status: "active"},
		{PlatformModelID: platformModel.ID, UpstreamModelID: activeModelB.ID, Protocol: "disabled_protocol", Status: "inactive"},
	}
	if err := db.Create(&routes).Error; err != nil {
		t.Fatalf("create routes: %v", err)
	}

	items, total, err := NewRepo(db).ListModels(ctx, repository.ListChannelModelsInput{
		Limit: 10,
		Sort:  "sortOrder_asc",
	})
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].SourceCount != 6 {
		t.Fatalf("expected source count 6, got %d", items[0].SourceCount)
	}
	if items[0].ActiveSourceCount != 3 {
		t.Fatalf("expected active source count 3, got %d", items[0].ActiveSourceCount)
	}
	assertProtocolsJSON(t, items[0].ProtocolsJSON, []string{"openai_responses", "xai_responses"})
	assertProtocolsJSON(t, items[1].ProtocolsJSON, []string{})
}

func TestListModelsSQLiteSortOrderKeepsVendorGroups(t *testing.T) {
	db := openChannelSQLiteTestDB(t)
	ctx := context.Background()

	models := []model.LLMPlatformModel{
		{Name: "claude-sonnet-4.6", Vendor: "anthropic", Status: "active", SortOrder: 100},
		{Name: "gpt-5.5", Vendor: "openai", Status: "active", SortOrder: 200},
		{Name: "gemini-3.1-pro", Vendor: "google", Status: "active", SortOrder: 300},
		{Name: "grok-4.3", Vendor: "xai", Status: "active", SortOrder: 400},
		{Name: "claude-fable-5", Vendor: "anthropic", Status: "active", SortOrder: 1000},
	}
	if err := db.Create(&models).Error; err != nil {
		t.Fatalf("create platform models: %v", err)
	}

	items, total, err := NewRepo(db).ListModels(ctx, repository.ListChannelModelsInput{
		Limit: 10,
		Sort:  "sortOrder_asc",
	})
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if total != int64(len(models)) {
		t.Fatalf("expected total %d, got %d", len(models), total)
	}
	got := modelNames(items)
	want := []string{
		"claude-sonnet-4.6",
		"claude-fable-5",
		"gpt-5.5",
		"gemini-3.1-pro",
		"grok-4.3",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected model order %v, got %v", want, got)
	}
}

func TestListModelsSQLiteSortOrderUsesAllModelsForVendorGroupAnchor(t *testing.T) {
	db := openChannelSQLiteTestDB(t)
	ctx := context.Background()

	models := []model.LLMPlatformModel{
		{Name: "claude-sonnet-4.6", Vendor: "anthropic", Status: "inactive", SortOrder: 100},
		{Name: "gpt-5.5", Vendor: "openai", Status: "active", SortOrder: 200},
		{Name: "gemini-3.1-pro", Vendor: "google", Status: "active", SortOrder: 300},
		{Name: "claude-fable-5", Vendor: "anthropic", Status: "active", SortOrder: 1000},
	}
	if err := db.Create(&models).Error; err != nil {
		t.Fatalf("create platform models: %v", err)
	}

	items, _, err := NewRepo(db).ListModels(ctx, repository.ListChannelModelsInput{
		Limit:      10,
		OnlyActive: true,
		Sort:       "sortOrder_asc",
	})
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	got := modelNames(items)
	want := []string{
		"claude-fable-5",
		"gpt-5.5",
		"gemini-3.1-pro",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected model order %v, got %v", want, got)
	}
}

func TestReorderModelsSQLiteUsesVendorGroupDisplayOrder(t *testing.T) {
	db := openChannelSQLiteTestDB(t)
	ctx := context.Background()

	models := []model.LLMPlatformModel{
		{Name: "claude-sonnet-4.6", Vendor: "anthropic", Status: "active", SortOrder: 100},
		{Name: "gpt-5.5", Vendor: "openai", Status: "active", SortOrder: 200},
		{Name: "gemini-3.1-pro", Vendor: "google", Status: "active", SortOrder: 300},
		{Name: "claude-fable-5", Vendor: "anthropic", Status: "active", SortOrder: 1000},
	}
	if err := db.Create(&models).Error; err != nil {
		t.Fatalf("create platform models: %v", err)
	}

	repo := NewRepo(db)
	if err := repo.ReorderModels(ctx, []uint{models[1].ID, models[0].ID, models[3].ID, models[2].ID}); err != nil {
		t.Fatalf("ReorderModels() error = %v", err)
	}
	items, _, err := repo.ListModels(ctx, repository.ListChannelModelsInput{
		Limit: 10,
		Sort:  "sortOrder_asc",
	})
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	got := modelNames(items)
	want := []string{
		"gpt-5.5",
		"claude-sonnet-4.6",
		"claude-fable-5",
		"gemini-3.1-pro",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected model order %v, got %v", want, got)
	}
}

func openChannelSQLiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := db.AutoMigrate(
		&model.LLMUpstream{},
		&model.LLMUpstreamModel{},
		&model.LLMPlatformModel{},
		&model.LLMPlatformModelRoute{},
	); err != nil {
		t.Fatalf("migrate channel tables: %v", err)
	}
	return db
}

func modelNames(items []ModelListRow) []string {
	results := make([]string, 0, len(items))
	for _, item := range items {
		results = append(results, item.PlatformModelName)
	}
	return results
}

func assertProtocolsJSON(t *testing.T, raw string, expected []string) {
	t.Helper()

	var actual []string
	if err := json.Unmarshal([]byte(raw), &actual); err != nil {
		t.Fatalf("unmarshal protocols JSON %q: %v", raw, err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected protocols %v, got %v", expected, actual)
	}
}
