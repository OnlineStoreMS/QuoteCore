package productcore

import "testing"

func TestSpecValuesLabel(t *testing.T) {
	got := SpecValuesLabel(map[string]string{"颜色分类": "盒装 HG400-9飞轮 11-34T"}, "颜色分类: 盒装 HG400-9飞轮 11-34T", "SKU1")
	want := "盒装 HG400-9飞轮 11-34T"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	got = SpecValuesLabel(map[string]string{"颜色": "红", "尺码": "L"}, "", "")
	if got != "L / 红" {
		t.Fatalf("multi got %q want %q", got, "L / 红")
	}

	got = SpecValuesLabel(nil, "颜色分类: 盒装 HG400-9飞轮 11-34T", "SKU1")
	if got != want {
		t.Fatalf("fallback got %q want %q", got, want)
	}
}
