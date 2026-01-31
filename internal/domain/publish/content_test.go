package publish

import (
	"reflect"
	"testing"
)

func TestImageContent_HasLocationAndMarkerTagsFields(t *testing.T) {
	typ := reflect.TypeOf(ImageContent{})
	if _, ok := typ.FieldByName("Location"); !ok {
		t.Fatalf("missing Location field")
	}
	if _, ok := typ.FieldByName("MarkerTags"); !ok {
		t.Fatalf("missing MarkerTags field")
	}
}

func TestFilterMarkerTags(t *testing.T) {
	t.Run("filters empty and whitespace", func(t *testing.T) {
		got := FilterMarkerTags([]string{"", "  ", "\t", "\n"})
		if got != nil {
			t.Fatalf("expected nil, got %#v", got)
		}
	})

	t.Run("keeps non-empty values and order", func(t *testing.T) {
		got := FilterMarkerTags([]string{"  深圳湾 ", "", "A", "  ", "B"})
		want := []string{"  深圳湾 ", "A", "B"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected result: got %#v, want %#v", got, want)
		}
	})
}
