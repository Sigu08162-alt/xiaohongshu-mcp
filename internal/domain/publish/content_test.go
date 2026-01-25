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
