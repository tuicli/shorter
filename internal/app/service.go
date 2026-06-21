package app

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tuicli/shorter/internal/domain"
)

type Repository interface {
	FindByOriginalURLs(ctx context.Context, urls []string) (map[string]domain.ShortLink, error)
	InsertLink(ctx context.Context, link domain.NewShortLink) (domain.ShortLink, error)
	ListLatest(ctx context.Context, page int, limit int) (RepositoryPage, error)
	Search(ctx context.Context, query string, page int, limit int) (RepositoryPage, error)
	GetByID(ctx context.Context, id int64) (domain.ShortLink, bool, error)
	GetByCode(ctx context.Context, code string) (domain.ShortLink, bool, error)
	SetStatus(ctx context.Context, id int64, status domain.LinkStatus, adminID int64) (domain.ShortLink, bool, error)
	ExportLinks(ctx context.Context, limit int) ([]domain.ShortLink, error)
	RecordEvent(ctx context.Context, event Event) error
}

type RepositoryPage struct {
	Links []domain.ShortLink
	Total int
}

type Event struct {
	ShortLinkID     int64
	EventType       string
	ActorTelegramID int64
	Metadata        string
}

type Options struct {
	BaseURL               string
	AddLinksMaxLines      int
	LinksPageSize         int
	CodeGenerationRetries int
	CSVExportMaxRows      int
}

type Service struct {
	repo                  Repository
	codegen               domain.CodeGenerator
	baseURL               string
	addLinksMaxLines      int
	linksPageSize         int
	codeGenerationRetries int
	csvExportMaxRows      int
}

func NewService(repo Repository, codegen domain.CodeGenerator, options Options) *Service {
	if codegen == nil {
		codegen = domain.RandomCodeGenerator{Length: 8}
	}
	if options.AddLinksMaxLines <= 0 {
		options.AddLinksMaxLines = 200
	}
	if options.LinksPageSize <= 0 {
		options.LinksPageSize = 10
	}
	if options.CodeGenerationRetries <= 0 {
		options.CodeGenerationRetries = 20
	}
	if options.CSVExportMaxRows <= 0 {
		options.CSVExportMaxRows = 10000
	}

	return &Service{
		repo:                  repo,
		codegen:               codegen,
		baseURL:               strings.TrimRight(options.BaseURL, "/"),
		addLinksMaxLines:      options.AddLinksMaxLines,
		linksPageSize:         options.LinksPageSize,
		codeGenerationRetries: options.CodeGenerationRetries,
		csvExportMaxRows:      options.CSVExportMaxRows,
	}
}

type PreviewNewRow struct {
	Line        int
	OriginalURL string
	Title       string
	Warnings    []string
}

type PreviewDuplicateRow struct {
	Line             int
	OriginalURL      string
	Kind             string
	FirstLine        int
	ExistingCode     string
	ExistingShortURL string
	ExistingStatus   domain.LinkStatus
}

type BulkPreview struct {
	NewRows         []PreviewNewRow
	DuplicateRows   []PreviewDuplicateRow
	InvalidLines    []domain.InvalidLine
	TotalInputLines int
}

func (p BulkPreview) HasNewRows() bool {
	return len(p.NewRows) > 0
}

func (s *Service) PreviewBulkLinks(ctx context.Context, text string) (BulkPreview, error) {
	parsed, err := domain.ParseBulkLinks(text, s.addLinksMaxLines)
	if err != nil {
		return BulkPreview{}, err
	}

	urls := make([]string, 0, len(parsed.Rows))
	for _, row := range parsed.Rows {
		urls = append(urls, row.OriginalURL)
	}

	existing, err := s.repo.FindByOriginalURLs(ctx, urls)
	if err != nil {
		return BulkPreview{}, fmt.Errorf("find existing originals: %w", err)
	}

	preview := BulkPreview{
		NewRows:         []PreviewNewRow{},
		DuplicateRows:   []PreviewDuplicateRow{},
		InvalidLines:    parsed.InvalidLines,
		TotalInputLines: countLines(text),
	}
	for _, duplicate := range parsed.LocalDuplicates {
		preview.DuplicateRows = append(preview.DuplicateRows, PreviewDuplicateRow{
			Line:        duplicate.Line,
			OriginalURL: duplicate.OriginalURL,
			Kind:        "local",
			FirstLine:   duplicate.FirstLine,
		})
	}
	for _, row := range parsed.Rows {
		link, ok := existing[row.OriginalURL]
		if ok {
			preview.DuplicateRows = append(preview.DuplicateRows, PreviewDuplicateRow{
				Line:             row.Line,
				OriginalURL:      row.OriginalURL,
				Kind:             "existing",
				ExistingCode:     link.Code,
				ExistingShortURL: s.ShortURL(link.Code),
				ExistingStatus:   link.Status,
			})
			continue
		}

		preview.NewRows = append(preview.NewRows, PreviewNewRow{
			Line:        row.Line,
			OriginalURL: row.OriginalURL,
			Title:       row.Title,
			Warnings:    append([]string{}, row.Warnings...),
		})
	}

	return preview, nil
}

type CreatedLink struct {
	Link     domain.ShortLink
	ShortURL string
}

type SkippedDuplicate struct {
	Line        int
	OriginalURL string
	Existing    domain.ShortLink
}

type CreateFailure struct {
	Line  int
	Error string
}

type CreateBulkResult struct {
	Created           []CreatedLink
	SkippedDuplicates []SkippedDuplicate
	Failures          []CreateFailure
}

func (s *Service) CreateBulkLinks(ctx context.Context, adminID int64, preview BulkPreview) (CreateBulkResult, error) {
	result := CreateBulkResult{
		Created:           []CreatedLink{},
		SkippedDuplicates: []SkippedDuplicate{},
		Failures:          []CreateFailure{},
	}
	if len(preview.NewRows) == 0 {
		return result, nil
	}

	urls := make([]string, 0, len(preview.NewRows))
	for _, row := range preview.NewRows {
		urls = append(urls, row.OriginalURL)
	}

	existing, err := s.repo.FindByOriginalURLs(ctx, urls)
	if err != nil {
		return CreateBulkResult{}, fmt.Errorf("recheck existing originals: %w", err)
	}

	for _, row := range preview.NewRows {
		if existingLink, ok := existing[row.OriginalURL]; ok {
			result.SkippedDuplicates = append(result.SkippedDuplicates, SkippedDuplicate{
				Line:        row.Line,
				OriginalURL: row.OriginalURL,
				Existing:    existingLink,
			})
			continue
		}

		created, err := s.insertWithCodeRetry(ctx, adminID, row)
		if err != nil {
			if errors.Is(err, domain.ErrOriginalURLExists) {
				result.SkippedDuplicates = append(result.SkippedDuplicates, SkippedDuplicate{
					Line:        row.Line,
					OriginalURL: row.OriginalURL,
				})
				continue
			}
			result.Failures = append(result.Failures, CreateFailure{
				Line:  row.Line,
				Error: err.Error(),
			})
			continue
		}

		result.Created = append(result.Created, CreatedLink{
			Link:     created,
			ShortURL: s.ShortURL(created.Code),
		})
	}

	return result, nil
}

func (s *Service) insertWithCodeRetry(ctx context.Context, adminID int64, row PreviewNewRow) (domain.ShortLink, error) {
	for attempt := 0; attempt < s.codeGenerationRetries; attempt++ {
		code, err := s.codegen.GenerateCode()
		if err != nil {
			return domain.ShortLink{}, fmt.Errorf("generate code: %w", err)
		}

		link, err := s.repo.InsertLink(ctx, domain.NewShortLink{
			Code:                code,
			OriginalURL:         row.OriginalURL,
			Title:               row.Title,
			CreatedByTelegramID: adminID,
		})
		if err == nil {
			return link, nil
		}
		if errors.Is(err, domain.ErrCodeExists) {
			continue
		}
		if errors.Is(err, domain.ErrOriginalURLExists) {
			return domain.ShortLink{}, err
		}
		return domain.ShortLink{}, fmt.Errorf("insert link: %w", err)
	}

	return domain.ShortLink{}, domain.ErrCodeCollisionLimit
}

type LinkView struct {
	Link     domain.ShortLink
	ShortURL string
}

type LinkPage struct {
	Links      []LinkView
	Total      int
	Page       int
	TotalPages int
}

func (s *Service) ListLatestLinks(ctx context.Context, page int) (LinkPage, error) {
	page = normalizePage(page)
	data, err := s.repo.ListLatest(ctx, page, s.linksPageSize)
	if err != nil {
		return LinkPage{}, fmt.Errorf("list latest links: %w", err)
	}
	return s.toLinkPage(data, page), nil
}

func (s *Service) SearchLinks(ctx context.Context, query string, page int) (LinkPage, error) {
	page = normalizePage(page)
	query = strings.TrimSpace(query)
	if query == "" {
		return LinkPage{Links: []LinkView{}, Page: page, TotalPages: 1}, nil
	}

	data, err := s.repo.Search(ctx, query, page, s.linksPageSize)
	if err != nil {
		return LinkPage{}, fmt.Errorf("search links: %w", err)
	}
	return s.toLinkPage(data, page), nil
}

func (s *Service) GetLink(ctx context.Context, id int64) (LinkView, bool, error) {
	link, ok, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return LinkView{}, false, fmt.Errorf("get link: %w", err)
	}
	if !ok || link.Status == domain.StatusDeleted {
		return LinkView{}, false, nil
	}
	return LinkView{Link: link, ShortURL: s.ShortURL(link.Code)}, true, nil
}

func (s *Service) DisableLink(ctx context.Context, id int64, adminID int64) (LinkView, bool, error) {
	return s.setStatus(ctx, id, domain.StatusDisabled, adminID)
}

func (s *Service) EnableLink(ctx context.Context, id int64, adminID int64) (LinkView, bool, error) {
	return s.setStatus(ctx, id, domain.StatusActive, adminID)
}

func (s *Service) DeleteLink(ctx context.Context, id int64, adminID int64) (LinkView, bool, error) {
	return s.setStatus(ctx, id, domain.StatusDeleted, adminID)
}

func (s *Service) setStatus(ctx context.Context, id int64, status domain.LinkStatus, adminID int64) (LinkView, bool, error) {
	link, ok, err := s.repo.SetStatus(ctx, id, status, adminID)
	if err != nil {
		return LinkView{}, false, fmt.Errorf("set link status: %w", err)
	}
	if !ok {
		return LinkView{}, false, nil
	}
	return LinkView{Link: link, ShortURL: s.ShortURL(link.Code)}, true, nil
}

type CSVExport struct {
	Filename string
	Content  []byte
	Capped   bool
	Rows     int
}

func (s *Service) ExportLinksCSV(ctx context.Context, adminID int64) (CSVExport, error) {
	links, err := s.repo.ExportLinks(ctx, s.csvExportMaxRows+1)
	if err != nil {
		return CSVExport{}, fmt.Errorf("export links: %w", err)
	}

	capped := len(links) > s.csvExportMaxRows
	if capped {
		links = links[:s.csvExportMaxRows]
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.Write([]string{"code", "short_url", "original_url", "title", "status", "created_at"}); err != nil {
		return CSVExport{}, fmt.Errorf("write csv header: %w", err)
	}
	for _, link := range links {
		if err := writer.Write([]string{
			link.Code,
			s.ShortURL(link.Code),
			link.OriginalURL,
			link.Title,
			string(link.Status),
			link.CreatedAt.Format(time.RFC3339),
		}); err != nil {
			return CSVExport{}, fmt.Errorf("write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return CSVExport{}, fmt.Errorf("flush csv: %w", err)
	}

	_ = s.repo.RecordEvent(ctx, Event{
		EventType:       "csv_exported",
		ActorTelegramID: adminID,
		Metadata:        `{"rows":` + strconv.Itoa(len(links)) + `,"capped":` + strconv.FormatBool(capped) + `}`,
	})

	suffix := time.Now().UTC().Format("20060102-150405")
	filename := "shorter-links-" + suffix + ".csv"
	if capped {
		filename = "shorter-links-capped-" + suffix + ".csv"
	}

	return CSVExport{
		Filename: filename,
		Content:  buf.Bytes(),
		Capped:   capped,
		Rows:     len(links),
	}, nil
}

type RedirectStatus string

const (
	RedirectFound    RedirectStatus = "redirect"
	RedirectInvalid  RedirectStatus = "invalid"
	RedirectNotFound RedirectStatus = "not_found"
	RedirectDisabled RedirectStatus = "disabled"
	RedirectDeleted  RedirectStatus = "deleted"
)

type RedirectResult struct {
	Status      RedirectStatus
	HTTPStatus  int
	OriginalURL string
	ShortLinkID int64
}

func (s *Service) ResolveRedirect(ctx context.Context, code string) (RedirectResult, error) {
	if !domain.IsValidCode(code) {
		return RedirectResult{Status: RedirectInvalid, HTTPStatus: 404}, nil
	}

	link, ok, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return RedirectResult{}, fmt.Errorf("resolve code: %w", err)
	}
	if !ok {
		return RedirectResult{Status: RedirectNotFound, HTTPStatus: 404}, nil
	}

	switch link.Status {
	case domain.StatusActive:
		return RedirectResult{
			Status:      RedirectFound,
			HTTPStatus:  302,
			OriginalURL: link.OriginalURL,
			ShortLinkID: link.ID,
		}, nil
	case domain.StatusDisabled:
		return RedirectResult{Status: RedirectDisabled, HTTPStatus: 404, ShortLinkID: link.ID}, nil
	case domain.StatusDeleted:
		return RedirectResult{Status: RedirectDeleted, HTTPStatus: 404, ShortLinkID: link.ID}, nil
	default:
		return RedirectResult{Status: RedirectNotFound, HTTPStatus: 404, ShortLinkID: link.ID}, nil
	}
}

func (s *Service) ShortURL(code string) string {
	return domain.BuildShortURL(s.baseURL, code)
}

func (s *Service) toLinkPage(data RepositoryPage, page int) LinkPage {
	links := make([]LinkView, 0, len(data.Links))
	for _, link := range data.Links {
		links = append(links, LinkView{Link: link, ShortURL: s.ShortURL(link.Code)})
	}

	return LinkPage{
		Links:      links,
		Total:      data.Total,
		Page:       page,
		TotalPages: totalPages(data.Total, s.linksPageSize),
	}
}

func normalizePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func totalPages(total int, limit int) int {
	if total <= 0 {
		return 1
	}
	pages := total / limit
	if total%limit != 0 {
		pages++
	}
	return pages
}

func countLines(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}
