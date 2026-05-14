package conversation

import (
	"strings"
	"testing"
)

const testUUID = "abcdef01-2345-6789-abcd-ef0123456789"
const testUUID2 = "11111111-2222-3333-4444-555555555555"

func TestImgSrcUploadsPattern(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCount int
		wantUUIDs []string
	}{
		// Happy paths.
		{
			name:      "relative_url",
			body:      `<img src="/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "absolute_url",
			body:      `<img src="https://libredesk.example.com/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "absolute_url_with_port",
			body:      `<img src="http://localhost:9000/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "with_query_string",
			body:      `<img src="/uploads/` + testUUID + `?sig=abc&exp=123">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "with_html_entity_query",
			body:      `<img src="/uploads/` + testUUID + `?sig=abc&amp;exp=123">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "single_quotes",
			body:      `<img src='/uploads/` + testUUID + `'>`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "attrs_before_src",
			body:      `<img class="inline-image" alt="x" data-foo="y" src="/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "attrs_after_src",
			body:      `<img src="/uploads/` + testUUID + `" class="x" alt="y">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "xhtml_self_closing",
			body:      `<img src="/uploads/` + testUUID + `" />`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "uppercase_img_tag",
			body:      `<IMG SRC="/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name: "multiline_tag",
			body: "<img\n  alt=\"x\"\n  src=\"/uploads/" + testUUID + "\"\n>",
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name: "multiple_in_body",
			body: `hello <img src="/uploads/` + testUUID + `"> world ` +
				`<img src="/uploads/` + testUUID2 + `">`,
			wantCount: 2,
			wantUUIDs: []string{testUUID, testUUID2},
		},

		// (?i) makes hex class case-insensitive too.
		{
			name:      "quirk_uppercase_hex_uuid_matches",
			body:      `<img src="/uploads/ABCDEF01-2345-6789-ABCD-EF0123456789">`,
			wantCount: 1,
			wantUUIDs: []string{"ABCDEF01-2345-6789-ABCD-EF0123456789"},
		},
		// `\b` boundary lets data-src match; harmless, no real src to render.
		{
			name:      "quirk_data_src_attribute_matches",
			body:      `<img alt="x" data-src="/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		// Not context-aware: comments are matched too.
		{
			name:      "quirk_inside_html_comment_matches",
			body:      `<!-- <img src="/uploads/` + testUUID + `"> -->`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},

		// Non-matches.
		{
			name:      "anchor_href_no_match",
			body:      `<a href="/uploads/` + testUUID + `">link</a>`,
			wantCount: 0,
		},
		{
			name:      "picture_source_no_match",
			body:      `<picture><source srcset="/uploads/` + testUUID + `"></picture>`,
			wantCount: 0,
		},
		{
			name:      "malformed_uuid_no_match",
			body:      `<img src="/uploads/not-a-uuid">`,
			wantCount: 0,
		},
		{
			name:      "uploads_filename_no_uuid_no_match",
			body:      `<img src="/uploads/photo.png">`,
			wantCount: 0,
		},
		{
			name:      "uuid_too_short_no_match",
			body:      `<img src="/uploads/abcdef01-2345-6789-abcd-ef0123">`,
			wantCount: 0,
		},
		{
			name:      "trailing_path_segment_no_match",
			body:      `<img src="/uploads/` + testUUID + `/extra">`,
			wantCount: 0,
		},
		{
			name:      "empty_src_no_match",
			body:      `<img src="">`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := imgSrcUploadsPattern.FindAllStringSubmatch(tt.body, -1)
			if len(matches) != tt.wantCount {
				t.Fatalf("match count = %d, want %d (matches=%v)", len(matches), tt.wantCount, matches)
			}
			for i, want := range tt.wantUUIDs {
				if matches[i][2] != want {
					t.Errorf("match %d uuid = %q, want %q", i, matches[i][2], want)
				}
			}
		})
	}
}


func TestImgSrcUploadsPattern_Adversarial(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCount int
	}{
		{
			name:      "image_element_should_not_match",
			body:      `<image src="/uploads/` + testUUID + `">`,
			wantCount: 0,
		},
		{
			name:      "imgblah_tag_should_not_match",
			body:      `<imgblah src="/uploads/` + testUUID + `">`,
			wantCount: 0,
		},
		{
			name:      "src_keyword_inside_alt_value_should_not_match",
			body:      `<img alt="see src=/uploads/foo" data-foo="bar">`,
			wantCount: 0,
		},
		{
			name:      "input_element_should_not_match",
			body:      `<input src="/uploads/` + testUUID + `">`,
			wantCount: 0,
		},
		{
			name:      "multiline_img_src_should_match",
			body:      "<img\n\tsrc=\"/uploads/" + testUUID + "\"\n>",
			wantCount: 1,
		},
		{
			name:      "extra_trailing_attributes_should_match",
			body:      `<img src="/uploads/` + testUUID + `" width="100" height="50" loading="lazy">`,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := imgSrcUploadsPattern.FindAllStringSubmatch(tt.body, -1)
			if len(matches) != tt.wantCount {
				t.Errorf("got %d matches, want %d\nbody: %s\nmatches: %v",
					len(matches), tt.wantCount, tt.body, matches)
			}
		})
	}
}

func TestExtractInlineImageUUIDs(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "empty_body",
			body: "",
			want: []string{},
		},
		{
			name: "no_images",
			body: "Just some text, no images here.",
			want: []string{},
		},
		{
			name: "single_image",
			body: `<img src="/uploads/` + testUUID + `">`,
			want: []string{testUUID},
		},
		{
			name: "two_distinct_images",
			body: `<img src="/uploads/` + testUUID + `"><img src="/uploads/` + testUUID2 + `">`,
			want: []string{testUUID, testUUID2},
		},
		{
			name: "duplicate_uuid_deduped",
			body: `<img src="/uploads/` + testUUID + `"><img src="/uploads/` + testUUID + `?v=2">`,
			want: []string{testUUID},
		},
		{
			name: "ignores_cid_form",
			body: `<img src="cid:ldsk-` + testUUID + `">`,
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInlineImageUUIDs(tt.body)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRewriteInlineImagesToCID(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "empty_body",
			body: "",
			want: "",
		},
		{
			name: "no_change_when_no_uploads",
			body: `<p>hello world</p>`,
			want: `<p>hello world</p>`,
		},
		{
			name: "single_relative",
			body: `<img src="/uploads/` + testUUID + `">`,
			want: `<img src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "absolute_url_with_query",
			body: `<img src="https://host.example.com/uploads/` + testUUID + `?sig=abc&exp=1">`,
			want: `<img src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "preserves_other_attributes",
			body: `<img class="inline-image" alt="hi" src="/uploads/` + testUUID + `">`,
			want: `<img class="inline-image" alt="hi" src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "preserves_single_quotes",
			body: `<img src='/uploads/` + testUUID + `'>`,
			want: `<img src='cid:ldsk-` + testUUID + `'>`,
		},
		{
			name: "rewrites_multiple",
			body: `<img src="/uploads/` + testUUID + `"><img src="/uploads/` + testUUID2 + `">`,
			want: `<img src="cid:ldsk-` + testUUID + `"><img src="cid:ldsk-` + testUUID2 + `">`,
		},
		{
			name: "leaves_cid_form_alone",
			body: `<img src="cid:ldsk-` + testUUID + `">`,
			want: `<img src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "leaves_non_uploads_alone",
			body: `<a href="/uploads/` + testUUID + `">link</a>`,
			want: `<a href="/uploads/` + testUUID + `">link</a>`,
		},
		{
			name: "is_idempotent",
			body: `<img src="cid:ldsk-` + testUUID + `">`,
			want: `<img src="cid:ldsk-` + testUUID + `">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteInlineImagesToCID(tt.body)
			if got != tt.want {
				t.Errorf("\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}

	// Round-trip: extract from URL form, rewrite, then extract again should
	// produce zero URL-form matches (only cid-form references remain).
	t.Run("round_trip_url_to_cid", func(t *testing.T) {
		body := `<img src="/uploads/` + testUUID + `">`
		rewritten := rewriteInlineImagesToCID(body)
		if strings.Contains(rewritten, "/uploads/") {
			t.Errorf("rewritten body still contains /uploads/: %s", rewritten)
		}
		leftover := extractInlineImageUUIDs(rewritten)
		if len(leftover) != 0 {
			t.Errorf("expected 0 URL-form UUIDs after rewrite, got %v", leftover)
		}
	})
}
