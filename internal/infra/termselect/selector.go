package termselect

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gookit/cliui/interact"
	"github.com/gookit/cliui/interact/ui"
)

type Item struct {
	Key    string
	Label  string
	Value  string
	Detail string
}

type Options struct {
	Title        string
	Items        []Item
	PageSize     int
	FilterPrompt string
}

type Selector interface {
	SelectMulti(ctx context.Context, opts Options) ([]Item, error)
}

type CliUISelector struct {
	backend interact.UIBackend
}

func NewCliUISelector() *CliUISelector {
	return NewCliUISelectorWithBackend(interact.NewUIReadlineBackend())
}

func NewCliUISelectorWithBackend(be interact.UIBackend) *CliUISelector {
	if be == nil {
		be = interact.NewUIReadlineBackend()
	}
	return &CliUISelector{backend: be}
}

func (s *CliUISelector) SelectMulti(ctx context.Context, opts Options) ([]Item, error) {
	if len(opts.Items) == 0 {
		return nil, nil
	}

	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "Select items"
	}

	uiItems := make([]ui.Item, 0, len(opts.Items))
	for idx, item := range opts.Items {
		item = normalizeItem(idx, item)
		uiItems = append(uiItems, ui.Item{
			Key:   item.Key,
			Label: itemLabel(item),
			Value: item,
		})
	}

	component := interact.NewUIMultiSelect(title, uiItems)
	component.Filterable = true
	component.FilterPrompt = opts.FilterPrompt
	component.PageSize = opts.PageSize
	if component.PageSize <= 0 {
		component.PageSize = 12
	}

	result, err := component.Run(ctx, s.backend)
	if err != nil {
		return nil, err
	}

	selected := make([]Item, 0, len(result.Values))
	for _, value := range result.Values {
		item, ok := value.(Item)
		if !ok {
			return nil, fmt.Errorf("unexpected selected item type %T", value)
		}
		selected = append(selected, item)
	}
	return selected, nil
}

func normalizeItem(idx int, item Item) Item {
	if strings.TrimSpace(item.Key) == "" {
		item.Key = strconv.Itoa(idx + 1)
	}
	if strings.TrimSpace(item.Label) == "" {
		item.Label = item.Value
	}
	return item
}

func itemLabel(item Item) string {
	if item.Detail == "" {
		return item.Label
	}
	return item.Label + " " + item.Detail
}
