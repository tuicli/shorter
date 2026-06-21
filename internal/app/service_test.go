package app

import (
	"context"
	"encoding/csv"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tuicli/shorter/internal/domain"
)

func TestPreviewAndCreateSkipsDuplicates(t *testing.T) {
	repo := newFakeRepo()
	existing := repo.mustInsert(domain.NewShortLink{
		Code:                "EXIST1",
		OriginalURL:         "https://example.com/existing",
		Title:               "existing",
		CreatedByTelegramID: 1,
	})
	service := NewService(repo, &sequenceCodegen{codes: []string{"NEW001"}}, Options{
		BaseURL:               "https://s.example",
		AddLinksMaxLines:      20,
		LinksPageSize:         10,
		CodeGenerationRetries: 3,
		CSVExportMaxRows:      100,
	})

	preview, err := service.PreviewBulkLinks(context.Background(), strings.Join([]string{
		"https://example.com/new first",
		"https://example.com/existing duplicate",
		"https://example.com/new repeated",
		"bad",
	}, "\n"))
	if err != nil {
		t.Fatalf("PreviewBulkLinks returned error: %v", err)
	}
	if len(preview.NewRows) != 1 {
		t.Fatalf("new rows = %d, want 1", len(preview.NewRows))
	}
	if len(preview.DuplicateRows) != 2 {
		t.Fatalf("duplicate rows = %d, want 2", len(preview.DuplicateRows))
	}
	if preview.DuplicateRows[0].Kind != "local" {
		t.Fatalf("first duplicate kind = %q", preview.DuplicateRows[0].Kind)
	}
	if preview.DuplicateRows[1].ExistingCode != existing.Code {
		t.Fatalf("existing duplicate code = %q", preview.DuplicateRows[1].ExistingCode)
	}
	if len(preview.InvalidLines) != 1 {
		t.Fatalf("invalid lines = %d, want 1", len(preview.InvalidLines))
	}

	result, err := service.CreateBulkLinks(context.Background(), 99, preview)
	if err != nil {
		t.Fatalf("CreateBulkLinks returned error: %v", err)
	}
	if len(result.Created) != 1 {
		t.Fatalf("created = %d, want 1", len(result.Created))
	}
	if result.Created[0].ShortURL != "https://s.example/NEW001" {
		t.Fatalf("short url = %q", result.Created[0].ShortURL)
	}
}

func TestCreateBulkLinksRetriesCodeCollision(t *testing.T) {
	repo := newFakeRepo()
	repo.mustInsert(domain.NewShortLink{
		Code:                "COLLID",
		OriginalURL:         "https://example.com/old",
		Title:               "old",
		CreatedByTelegramID: 1,
	})
	service := NewService(repo, &sequenceCodegen{codes: []string{"COLLID", "UNIQUE"}}, Options{
		BaseURL:               "https://s.example",
		CodeGenerationRetries: 2,
	})

	result, err := service.CreateBulkLinks(context.Background(), 99, BulkPreview{
		NewRows: []PreviewNewRow{{
			Line:        1,
			OriginalURL: "https://example.com/new",
			Title:       "new",
		}},
	})
	if err != nil {
		t.Fatalf("CreateBulkLinks returned error: %v", err)
	}
	if len(result.Created) != 1 || result.Created[0].Link.Code != "UNIQUE" {
		t.Fatalf("created = %#v, want UNIQUE", result.Created)
	}
}

func TestCreateBulkLinksRechecksStalePreviewDuplicates(t *testing.T) {
	repo := newFakeRepo()
	service := NewService(repo, &sequenceCodegen{codes: []string{"NEW001"}}, Options{
		BaseURL:               "https://s.example",
		CodeGenerationRetries: 3,
	})

	preview, err := service.PreviewBulkLinks(context.Background(), "https://example.com/new title")
	if err != nil {
		t.Fatalf("PreviewBulkLinks returned error: %v", err)
	}
	repo.mustInsert(domain.NewShortLink{
		Code:                "OTHER1",
		OriginalURL:         "https://example.com/new",
		Title:               "other",
		CreatedByTelegramID: 1,
	})

	result, err := service.CreateBulkLinks(context.Background(), 99, preview)
	if err != nil {
		t.Fatalf("CreateBulkLinks returned error: %v", err)
	}
	if len(result.Created) != 0 {
		t.Fatalf("created = %d, want 0", len(result.Created))
	}
	if len(result.SkippedDuplicates) != 1 {
		t.Fatalf("skipped duplicates = %d, want 1", len(result.SkippedDuplicates))
	}
}

func TestStatusLifecycleAffectsRedirectAndLists(t *testing.T) {
	repo := newFakeRepo()
	link := repo.mustInsert(domain.NewShortLink{
		Code:                "ABC123",
		OriginalURL:         "https://example.com/a",
		Title:               "a",
		CreatedByTelegramID: 1,
	})
	service := NewService(repo, nil, Options{BaseURL: "https://s.example"})

	redirect, err := service.ResolveRedirect(context.Background(), "ABC123")
	if err != nil {
		t.Fatalf("ResolveRedirect active error: %v", err)
	}
	if redirect.Status != RedirectFound || redirect.OriginalURL != link.OriginalURL {
		t.Fatalf("active redirect = %#v", redirect)
	}

	if _, ok, err := service.DisableLink(context.Background(), link.ID, 99); err != nil || !ok {
		t.Fatalf("DisableLink ok=%v err=%v", ok, err)
	}
	redirect, err = service.ResolveRedirect(context.Background(), "ABC123")
	if err != nil {
		t.Fatalf("ResolveRedirect disabled error: %v", err)
	}
	if redirect.Status != RedirectDisabled {
		t.Fatalf("disabled redirect = %#v", redirect)
	}

	if _, ok, err := service.EnableLink(context.Background(), link.ID, 99); err != nil || !ok {
		t.Fatalf("EnableLink ok=%v err=%v", ok, err)
	}
	redirect, err = service.ResolveRedirect(context.Background(), "ABC123")
	if err != nil {
		t.Fatalf("ResolveRedirect enabled error: %v", err)
	}
	if redirect.Status != RedirectFound {
		t.Fatalf("enabled redirect = %#v", redirect)
	}

	if _, ok, err := service.DeleteLink(context.Background(), link.ID, 99); err != nil || !ok {
		t.Fatalf("DeleteLink ok=%v err=%v", ok, err)
	}
	redirect, err = service.ResolveRedirect(context.Background(), "ABC123")
	if err != nil {
		t.Fatalf("ResolveRedirect deleted error: %v", err)
	}
	if redirect.Status != RedirectDeleted {
		t.Fatalf("deleted redirect = %#v", redirect)
	}

	page, err := service.ListLatestLinks(context.Background(), 1, LinkSortTime)
	if err != nil {
		t.Fatalf("ListLatestLinks returned error: %v", err)
	}
	if len(page.Links) != 0 {
		t.Fatalf("latest links = %d, want 0 after delete", len(page.Links))
	}
}

func TestListLatestLinksSortsByTitle(t *testing.T) {
	repo := newFakeRepo()
	repo.mustInsert(domain.NewShortLink{
		Code:                "CHARLIE",
		OriginalURL:         "https://example.com/c",
		Title:               "Charlie",
		CreatedByTelegramID: 1,
	})
	repo.mustInsert(domain.NewShortLink{
		Code:                "ALPHA",
		OriginalURL:         "https://example.com/a",
		Title:               "Alpha",
		CreatedByTelegramID: 1,
	})
	repo.mustInsert(domain.NewShortLink{
		Code:                "BETA",
		OriginalURL:         "https://example.com/b",
		Title:               "beta",
		CreatedByTelegramID: 1,
	})
	service := NewService(repo, nil, Options{BaseURL: "https://s.example", LinksPageSize: 10})

	page, err := service.ListLatestLinks(context.Background(), 1, LinkSortTitle)
	if err != nil {
		t.Fatalf("ListLatestLinks returned error: %v", err)
	}
	got := []string{}
	for _, item := range page.Links {
		got = append(got, item.Link.Title)
	}
	want := []string{"Alpha", "beta", "Charlie"}
	if !slices.Equal(got, want) {
		t.Fatalf("titles = %#v, want %#v", got, want)
	}
	if page.Sort != LinkSortTitle {
		t.Fatalf("sort = %q, want %q", page.Sort, LinkSortTitle)
	}
}

func TestExportLinksCSV(t *testing.T) {
	repo := newFakeRepo()
	repo.mustInsert(domain.NewShortLink{
		Code:                "CSV001",
		OriginalURL:         "https://example.com/a,b",
		Title:               "title \"quoted\"\nnext",
		CreatedByTelegramID: 1,
	})
	service := NewService(repo, nil, Options{BaseURL: "https://s.example", CSVExportMaxRows: 100})

	export, err := service.ExportLinksCSV(context.Background(), 99)
	if err != nil {
		t.Fatalf("ExportLinksCSV returned error: %v", err)
	}

	rows, err := csv.NewReader(strings.NewReader(string(export.Content))).ReadAll()
	if err != nil {
		t.Fatalf("csv read returned error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[1][1] != "https://s.example/CSV001" {
		t.Fatalf("short url = %q", rows[1][1])
	}
	if rows[1][3] != "title \"quoted\"\nnext" {
		t.Fatalf("title = %q", rows[1][3])
	}
}

type sequenceCodegen struct {
	codes []string
	index int
}

func (g *sequenceCodegen) GenerateCode() (string, error) {
	if g.index >= len(g.codes) {
		return "", errors.New("no more codes")
	}
	code := g.codes[g.index]
	g.index++
	return code, nil
}

type fakeRepo struct {
	nextID int64
	links  map[int64]domain.ShortLink
	events []Event
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		nextID: 1,
		links:  map[int64]domain.ShortLink{},
		events: []Event{},
	}
}

func (r *fakeRepo) FindByOriginalURLs(_ context.Context, urls []string) (map[string]domain.ShortLink, error) {
	allowed := map[string]struct{}{}
	for _, rawURL := range urls {
		allowed[rawURL] = struct{}{}
	}
	out := map[string]domain.ShortLink{}
	for _, link := range r.links {
		if _, ok := allowed[link.OriginalURL]; ok {
			out[link.OriginalURL] = link
		}
	}
	return out, nil
}

func (r *fakeRepo) InsertLink(_ context.Context, input domain.NewShortLink) (domain.ShortLink, error) {
	for _, link := range r.links {
		if link.Code == input.Code {
			return domain.ShortLink{}, domain.ErrCodeExists
		}
		if link.OriginalURL == input.OriginalURL {
			return domain.ShortLink{}, domain.ErrOriginalURLExists
		}
	}
	return r.mustInsert(input), nil
}

func (r *fakeRepo) mustInsert(input domain.NewShortLink) domain.ShortLink {
	now := time.Unix(1700000000+r.nextID, 0).UTC()
	link := domain.ShortLink{
		ID:                  r.nextID,
		Code:                input.Code,
		OriginalURL:         input.OriginalURL,
		Title:               input.Title,
		Status:              domain.StatusActive,
		CreatedByTelegramID: input.CreatedByTelegramID,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	r.links[link.ID] = link
	r.nextID++
	return link
}

func (r *fakeRepo) ListLatest(_ context.Context, options LinkListOptions) (RepositoryPage, error) {
	options = normalizeFakeListOptions(options)
	links := make([]domain.ShortLink, 0, len(r.links))
	for _, link := range r.links {
		if link.Status != domain.StatusDeleted {
			links = append(links, link)
		}
	}
	sortFakeLinks(links, options.Sort)
	return pageSlice(links, options.Page, options.Limit), nil
}

func (r *fakeRepo) Search(_ context.Context, query string, options LinkListOptions) (RepositoryPage, error) {
	options = normalizeFakeListOptions(options)
	query = strings.ToLower(query)
	links := []domain.ShortLink{}
	for _, link := range r.links {
		if link.Status == domain.StatusDeleted {
			continue
		}
		haystack := strings.ToLower(link.Code + " " + link.Title + " " + link.OriginalURL)
		if strings.Contains(haystack, query) {
			links = append(links, link)
		}
	}
	sortFakeLinks(links, options.Sort)
	return pageSlice(links, options.Page, options.Limit), nil
}

func (r *fakeRepo) GetByID(_ context.Context, id int64) (domain.ShortLink, bool, error) {
	link, ok := r.links[id]
	return link, ok, nil
}

func (r *fakeRepo) GetByCode(_ context.Context, code string) (domain.ShortLink, bool, error) {
	for _, link := range r.links {
		if link.Code == code {
			return link, true, nil
		}
	}
	return domain.ShortLink{}, false, nil
}

func (r *fakeRepo) SetStatus(_ context.Context, id int64, status domain.LinkStatus, adminID int64) (domain.ShortLink, bool, error) {
	link, ok := r.links[id]
	if !ok {
		return domain.ShortLink{}, false, nil
	}
	if !status.Valid() {
		return domain.ShortLink{}, false, domain.ErrInvalidStatus
	}
	now := time.Unix(1800000000, 0).UTC()
	link.Status = status
	link.UpdatedByTelegramID = adminID
	link.UpdatedAt = now
	switch status {
	case domain.StatusActive:
		link.DisabledAt = nil
		link.DisabledByTelegramID = 0
	case domain.StatusDisabled:
		link.DisabledAt = &now
		link.DisabledByTelegramID = adminID
	case domain.StatusDeleted:
		link.DeletedAt = &now
		link.DeletedByTelegramID = adminID
	}
	r.links[id] = link
	return link, true, nil
}

func (r *fakeRepo) ExportLinks(_ context.Context, limit int) ([]domain.ShortLink, error) {
	links := make([]domain.ShortLink, 0, len(r.links))
	for _, link := range r.links {
		links = append(links, link)
	}
	slices.SortFunc(links, func(a domain.ShortLink, b domain.ShortLink) int {
		return int(a.ID - b.ID)
	})
	if len(links) > limit {
		return links[:limit], nil
	}
	return links, nil
}

func (r *fakeRepo) RecordEvent(_ context.Context, event Event) error {
	r.events = append(r.events, event)
	return nil
}

func normalizeFakeListOptions(options LinkListOptions) LinkListOptions {
	if options.Page < 1 {
		options.Page = 1
	}
	if options.Limit < 1 {
		options.Limit = 10
	}
	options.Sort = options.Sort.Normalize()
	return options
}

func sortFakeLinks(links []domain.ShortLink, sort LinkSort) {
	switch sort.Normalize() {
	case LinkSortTitle:
		slices.SortFunc(links, func(a domain.ShortLink, b domain.ShortLink) int {
			left := strings.ToLower(a.Title)
			right := strings.ToLower(b.Title)
			if left == right {
				return compareIDDesc(a.ID, b.ID)
			}
			return strings.Compare(left, right)
		})
	default:
		slices.SortFunc(links, func(a domain.ShortLink, b domain.ShortLink) int {
			if a.CreatedAt.Equal(b.CreatedAt) {
				return compareIDDesc(a.ID, b.ID)
			}
			if a.CreatedAt.After(b.CreatedAt) {
				return -1
			}
			return 1
		})
	}
}

func compareIDDesc(left int64, right int64) int {
	if left == right {
		return 0
	}
	if left > right {
		return -1
	}
	return 1
}

func pageSlice(links []domain.ShortLink, page int, limit int) RepositoryPage {
	total := len(links)
	offset := (page - 1) * limit
	if offset >= len(links) {
		return RepositoryPage{Links: []domain.ShortLink{}, Total: total}
	}
	end := offset + limit
	if end > len(links) {
		end = len(links)
	}
	return RepositoryPage{Links: links[offset:end], Total: total}
}
