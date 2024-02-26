package converter

import (
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/widgets"
)

func KeyValueFromWidgetItems(items []*widgets.KeyValueItem) []domain.KeyValue {
	out := make([]domain.KeyValue, 0, len(items))
	for _, v := range items {
		out = append(out, domain.KeyValue{
			ID:     v.Identifier,
			Key:    v.Key,
			Value:  v.Value,
			Enable: v.Active,
		})
	}

	return out
}

func WidgetItemsFromKeyValue(items []domain.KeyValue) []*widgets.KeyValueItem {
	out := make([]*widgets.KeyValueItem, 0, len(items))
	for _, v := range items {
		out = append(out, widgets.NewKeyValueItem(v.Key, v.Value, v.ID, v.Enable))
	}

	return out
}
