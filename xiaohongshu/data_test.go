package xiaohongshu

import "testing"

func TestParseFeedsJSON_MapsTitleAndCover(t *testing.T) {
	input := `[{"id":"note1","user_id":"user1","title_candidates":["","  ","标题1"],"cover":"https://example.com/a.jpg","xsec_token":"tok"}]`

	feeds, err := parseFeedsJSON(input)
	if err != nil {
		t.Fatalf("parseFeedsJSON err: %v", err)
	}
	if len(feeds) != 1 {
		t.Fatalf("expected one feed")
	}
	if feeds[0].NoteCard.DisplayTitle != "标题1" {
		t.Fatalf("unexpected title: %q", feeds[0].NoteCard.DisplayTitle)
	}
	if feeds[0].NoteCard.Cover.URLDefault != "https://example.com/a.jpg" {
		t.Fatalf("unexpected cover: %q", feeds[0].NoteCard.Cover.URLDefault)
	}
	if feeds[0].XsecToken != "tok" {
		t.Fatalf("unexpected token: %q", feeds[0].XsecToken)
	}
}

func TestPickTitle_UsesFirstNonEmpty(t *testing.T) {
	title := pickTitle([]string{"", "  ", "标题A", "标题B"})
	if title != "标题A" {
		t.Fatalf("unexpected title: %q", title)
	}
}

func TestPickTitleFromLines_UsesFirstLine(t *testing.T) {
	title := pickTitleFromLines([]string{"标题1", "ZIIKOO TALK", "赞"})
	if title != "标题1" {
		t.Fatalf("unexpected title: %q", title)
	}
}

func TestParseFeedsJSON_MapsMetricsAndPublishTime(t *testing.T) {
	input := `[{"id":"note2","user_id":"user2","title_candidates":["标题2"],"cover":"https://example.com/b.jpg","xsec_token":"tok2","liked_count":"12","comment_count":"3","collected_count":"5","publish_time":"昨天"}]`

	feeds, err := parseFeedsJSON(input)
	if err != nil {
		t.Fatalf("parseFeedsJSON err: %v", err)
	}
	if len(feeds) != 1 {
		t.Fatalf("expected one feed")
	}
	if feeds[0].NoteCard.InteractInfo.LikedCount != "12" {
		t.Fatalf("unexpected liked count: %q", feeds[0].NoteCard.InteractInfo.LikedCount)
	}
	if feeds[0].NoteCard.InteractInfo.CommentCount != "3" {
		t.Fatalf("unexpected comment count: %q", feeds[0].NoteCard.InteractInfo.CommentCount)
	}
	if feeds[0].NoteCard.InteractInfo.CollectedCount != "5" {
		t.Fatalf("unexpected collected count: %q", feeds[0].NoteCard.InteractInfo.CollectedCount)
	}
	if feeds[0].NoteCard.PublishTime != "昨天" {
		t.Fatalf("unexpected publish time: %q", feeds[0].NoteCard.PublishTime)
	}
}

func TestNormalizeStateNotesRaw_ParsesVariants(t *testing.T) {
	raw := []any{
		map[string]any{
			"id":    "note1",
			"title": "标题1",
			"cover": map[string]any{"urlDefault": "https://example.com/a.jpg"},
		},
	}
	notes := normalizeStateNotesRaw(raw)
	if len(notes) != 1 {
		t.Fatalf("expected one note")
	}
	if notes[0].ID != "note1" {
		t.Fatalf("unexpected id: %q", notes[0].ID)
	}
	if notes[0].Title != "标题1" {
		t.Fatalf("unexpected title: %q", notes[0].Title)
	}
	if notes[0].Cover != "https://example.com/a.jpg" {
		t.Fatalf("unexpected cover: %q", notes[0].Cover)
	}

	rawValue := map[string]any{
		"_value": []any{
			map[string]any{
				"noteId":       "note2",
				"displayTitle": "标题2",
				"cover":        map[string]any{"url": "https://example.com/b.jpg"},
			},
		},
	}
	notes = normalizeStateNotesRaw(rawValue)
	if len(notes) != 1 || notes[0].ID != "note2" || notes[0].Title != "标题2" || notes[0].Cover != "https://example.com/b.jpg" {
		t.Fatalf("unexpected parsed note: %+v", notes)
	}

	rawList := map[string]any{
		"list": []any{
			map[string]any{
				"note_id":   "note3",
				"noteTitle": "标题3",
				"cover":     map[string]any{"urlDefault": "https://example.com/c.jpg"},
			},
		},
	}
	notes = normalizeStateNotesRaw(rawList)
	if len(notes) != 1 || notes[0].ID != "note3" || notes[0].Title != "标题3" || notes[0].Cover != "https://example.com/c.jpg" {
		t.Fatalf("unexpected parsed note: %+v", notes)
	}

	rawMap := map[string]any{
		"note4": map[string]any{
			"id":    "note4",
			"title": "标题4",
			"cover": map[string]any{"urlDefault": "https://example.com/d.jpg"},
		},
	}
	notes = normalizeStateNotesRaw(rawMap)
	if len(notes) != 1 || notes[0].ID != "note4" || notes[0].Title != "标题4" || notes[0].Cover != "https://example.com/d.jpg" {
		t.Fatalf("unexpected parsed note: %+v", notes)
	}
}

func TestBuildStateNoteMapsFromRaw(t *testing.T) {
	raw := []any{
		map[string]any{
			"id":    "note1",
			"title": "标题1",
			"cover": map[string]any{"urlDefault": "https://example.com/a.jpg"},
		},
	}
	titleMap, coverMap := buildStateNoteMapsFromRaw(raw)
	if titleMap["note1"] != "标题1" {
		t.Fatalf("unexpected title map: %q", titleMap["note1"])
	}
	if coverMap["note1"] != "https://example.com/a.jpg" {
		t.Fatalf("unexpected cover map: %q", coverMap["note1"])
	}
}
