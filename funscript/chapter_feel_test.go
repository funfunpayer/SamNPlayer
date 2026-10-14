package funscript

import "testing"

func TestSuggestChapterFeelsAIWins(t *testing.T) {
	chs := []ChapterMark{
		{Name: "A", StartTime: 0, EndTime: 10000},
		{Name: "B", StartTime: 10000, EndTime: 20000},
	}
	proposals := func(ms int64) (string, float64, bool) {
		if ms < 10000 {
			return "blowjob", 0.9, true
		}
		return "penetration", 0.7, true
	}
	got := SuggestChapterFeels(chs, proposals, "")
	if len(got) != 2 || got[0].Feel != SceneFeelOral || got[0].Source != "ai" || got[1].Feel != SceneFeelIntercourse {
		t.Fatalf("got %+v", got)
	}
}

func TestSuggestChapterFeelsContactFallback(t *testing.T) {
	chs := []ChapterMark{{StartTime: 0, EndTime: 5000}}
	got := SuggestChapterFeels(chs, nil, "mouth")
	if len(got) != 1 || got[0].Feel != SceneFeelOral || got[0].Source != "contact" {
		t.Fatalf("got %+v", got)
	}
}

func TestSuggestChapterFeelsNone(t *testing.T) {
	got := SuggestChapterFeels(nil, nil, "")
	if got != nil {
		t.Fatal(got)
	}
}
